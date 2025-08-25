package oneagent

import (
    "context"
    "fmt"

    "github.com/rkitindi-kr/dynatrace-bootstrapper/cmd"
    "github.com/rkitindi-kr/dynatrace-bootstrapper/cmd/configure"
    "github.com/rkitindi-kr/dynatrace-bootstrapper/cmd/move"
    "github.com/rkitindi-kr/dynatrace-operator/cmd/bootstrapper"
    "github.com/rkitindi-kr/dynatrace-operator/pkg/api/latest/dynakube"
    "github.com/rkitindi-kr/dynatrace-operator/pkg/consts"
    maputils "github.com/rkitindi-kr/dynatrace-operator/pkg/util/map"
    dtwebhook "github.com/rkitindi-kr/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/log"
)

// mutateInitContainer mutates the Pod's init container depending on whether CSI or PVC is used.
// It ensures PVC creation is immediate and mounts are properly configured.
func mutateInitContainer(mut *dtwebhook.Mutator, mutationRequest *dtwebhook.MutationRequest, installPath string) error {
    logger := log.FromContext(context.Background())
    ctx := context.Background() // local context for PVC creation

    if isCSIVolume(mutationRequest.BaseRequest) {
        logger.Info("Configuring init-container with CSI bin volume", "pod", mutationRequest.PodName())

        addCSIBinVolume(
            mutationRequest.Pod,
            mutationRequest.DynaKube.Name,
            mutationRequest.DynaKube.FF().GetCSIMaxRetryTimeout().String(),
        )

        // CSI volumes are readonly
        addInitBinMount(mutationRequest.InstallContainer, true)
    } else {
        logger.Info("Configuring init-container with PVC bin volume", "pod", mutationRequest.PodName())

        // Ensure PVC exists using client from mutator
        if mutClient, ok := mut.(*Mutator); ok && mutClient.Client != nil {
            if err := addPVCBinVolume(ctx, mutClient.Client, mutationRequest.Pod, "2Gi", "robin-repl-3"); err != nil {
                logger.Error(err, "Failed to create PVC bin volume")
                return err
            }
        } else {
            return fmt.Errorf("mutator client is nil, cannot create PVC")
        }

        // PVC-backed volume must be writable for init-container
        addInitBinMount(mutationRequest.InstallContainer, false)

        if mutationRequest.DynaKube.FF().IsNodeImagePull() {
            logger.Info("Configuring init-container with self-extracting image", "pod", mutationRequest.PodName())

            if len(mutationRequest.InstallContainer.Args) > 1 {
                mutationRequest.InstallContainer.Args = mutationRequest.InstallContainer.Args[1:]
            }
            mutationRequest.InstallContainer.Image = mutationRequest.DynaKube.OneAgent().GetCodeModulesImage()
        } else {
            logger.Info("Configuring init-container for ZIP download", "pod", mutationRequest.PodName())

            downloadArgs := []arg.Arg{
                {Name: bootstrapper.TargetVersionFlag, Value: mutationRequest.DynaKube.OneAgent().GetCodeModulesVersion()},
            }

            if flavor := maputils.GetField(mutationRequest.Pod.Annotations, AnnotationFlavor, ""); flavor != "" {
                downloadArgs = append(downloadArgs, arg.Arg{Name: bootstrapper.FlavorFlag, Value: flavor})
            }

            if mutationRequest.InstallContainer.Args == nil {
                mutationRequest.InstallContainer.Args = []string{}
            }

            mutationRequest.InstallContainer.Args = append(
                mutationRequest.InstallContainer.Args,
                arg.ConvertArgsToStrings(downloadArgs)...,
            )
        }
    }

    return addInitArgs(*mutationRequest.Pod, mutationRequest.InstallContainer, mutationRequest.DynaKube, installPath)
}

// addInitArgs appends the necessary arguments to the init-container
func addInitArgs(pod corev1.Pod, initContainer *corev1.Container, dk dynakube.DynaKube, installPath string) error {
    args := []arg.Arg{
        {Name: cmd.SourceFolderFlag, Value: AgentCodeModuleSource},
        {Name: cmd.TargetFolderFlag, Value: consts.AgentInitBinDirMount},
        {Name: configure.InstallPathFlag, Value: installPath},
    }

    if dk.OneAgent().IsCloudNativeFullstackMode() {
        tenantUUID, err := dk.TenantUUID()
        if err != nil {
            return err
        }

        args = append(args,
            arg.Arg{Name: configure.IsFullstackFlag},
            arg.Arg{Name: configure.TenantFlag, Value: tenantUUID},
        )
    }

    if technology := getTechnology(pod, dk); technology != "" {
        args = append(args, arg.Arg{Name: move.TechnologyFlag, Value: technology})
    }

    if initContainer.Args == nil {
        initContainer.Args = []string{}
    }

    initContainer.Args = append(initContainer.Args, arg.ConvertArgsToStrings(args)...)
    return nil
}

// getTechnology returns the technology annotation or fallback
func getTechnology(pod corev1.Pod, dk dynakube.DynaKube) string {
    return maputils.GetField(pod.Annotations, AnnotationTechnologies, dk.FF().GetNodeImagePullTechnology())
}

// Security helpers
func HasPodUserSet(ctx *corev1.PodSecurityContext) bool {
    return ctx != nil && ctx.RunAsUser != nil
}

func HasPodGroupSet(ctx *corev1.PodSecurityContext) bool {
    return ctx != nil && ctx.RunAsGroup != nil
}

func IsNonRoot(ctx *corev1.SecurityContext) bool {
    return ctx != nil &&
        (ctx.RunAsUser != nil && *ctx.RunAsUser != RootUserGroup) &&
        (ctx.RunAsGroup != nil && *ctx.RunAsGroup != RootUserGroup)
}

