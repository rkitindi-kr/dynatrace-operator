package oneagent

import (
    "context"
    "fmt"

    "github.com/rkitindi-kr/dynatrace-operator/pkg/api/latest/dynakube"
    "github.com/rkitindi-kr/dynatrace-operator/pkg/util/kubeobjects/mounts"
    maputils "github.com/rkitindi-kr/dynatrace-operator/pkg/util/map"
    dtwebhook "github.com/rkitindi-kr/dynatrace-operator/pkg/webhook/mutation/pod/mutator"
    corev1 "k8s.io/api/core/v1"
    "sigs.k8s.io/controller-runtime/pkg/client"
    "sigs.k8s.io/controller-runtime/pkg/log"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Mutator struct {
    Client client.Client
}

// Constructor
func NewMutator(c client.Client) dtwebhook.Mutator {
    return &Mutator{
        Client: c,
    }
}

// ================== Init Container Mutation ==================
func (mut *Mutator) mutateInitContainer(request *dtwebhook.MutationRequest, installPath string) error {
    logger := log.FromContext(context.Background())

    if request.Pod == nil {
        return fmt.Errorf("mutation request Pod is nil")
    }
    if request.InstallContainer == nil {
        return fmt.Errorf("mutation request InstallContainer is nil")
    }

    ctx := context.Background()

    if isCSIVolume(request.BaseRequest) {
        logger.Info("configuring init-container with CSI bin volume", "pod", request.PodName())
        addCSIBinVolume(
            request.Pod,
            request.DynaKube.Name,
            request.DynaKube.FF().GetCSIMaxRetryTimeout().String(),
        )
        addInitBinMount(request.InstallContainer, true)
    } else {
        logger.Info("configuring init-container with PVC bin volume", "pod", request.PodName())
        // Create PVC immediately
        if err := addPVCBinVolume(ctx, mut.Client, request.Pod, "2Gi", "robin-repl-3"); err != nil {
            logger.Error(err, "failed to add PVC bin volume", "pod", request.PodName())
            return fmt.Errorf("failed to add PVC bin volume for pod %s: %w", request.PodName(), err)
        }

        addInitBinMount(request.InstallContainer, false)

        // Configure container args/image depending on Dynakube feature flags
        if request.DynaKube.FF().IsNodeImagePull() {
            logger.Info("configuring init-container with self-extracting image", "pod", request.PodName())
            if len(request.InstallContainer.Args) > 1 {
                request.InstallContainer.Args = request.InstallContainer.Args[1:]
            }
            request.InstallContainer.Image = request.DynaKube.OneAgent().GetCodeModulesImage()
        } else {
            logger.Info("configuring init-container for ZIP download", "pod", request.PodName())
            downloadArgs := []arg.Arg{
                {Name: bootstrapper.TargetVersionFlag, Value: request.DynaKube.OneAgent().GetCodeModulesVersion()},
            }

            if flavor := maputils.GetField(request.Pod.Annotations, AnnotationFlavor, ""); flavor != "" {
                downloadArgs = append(downloadArgs, arg.Arg{Name: bootstrapper.FlavorFlag, Value: flavor})
            }

            request.InstallContainer.Args = append(request.InstallContainer.Args, arg.ConvertArgsToStrings(downloadArgs)...)
        }
    }

    return addInitArgs(*request.Pod, request.InstallContainer, request.DynaKube, installPath)
}

// ================== Mutator Interface Implementation ==================
func (mut *Mutator) IsEnabled(request *dtwebhook.BaseRequest) bool {
    enabledOnPod := maputils.GetFieldBool(request.Pod.Annotations, AnnotationInject, request.DynaKube.FF().IsAutomaticInjection())
    enabledOnDynakube := request.DynaKube.OneAgent().GetNamespaceSelector() != nil

    matchesNamespaceSelector := true
    if request.DynaKube.OneAgent().GetNamespaceSelector().Size() > 0 {
        selector, _ := metav1.LabelSelectorAsSelector(request.DynaKube.OneAgent().GetNamespaceSelector())
        matchesNamespaceSelector = selector.Matches(request.Namespace.Labels)
    }

    return matchesNamespaceSelector && enabledOnPod && enabledOnDynakube
}

func (mut *Mutator) IsInjected(request *dtwebhook.BaseRequest) bool {
    return maputils.GetFieldBool(request.Pod.Annotations, AnnotationInjected, false)
}

func (mut *Mutator) Mutate(request *dtwebhook.MutationRequest) error {
    if request.Pod == nil {
        return fmt.Errorf("mutation request Pod is nil")
    }

    installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)

    // Mutate init container (includes PVC creation)
    if err := mut.mutateInitContainer(request, installPath); err != nil {
        return err
    }

    // Mutate user containers
    _ = mutateUserContainers(request.BaseRequest, installPath)

    // Set injected annotation
    setInjectedAnnotation(request.Pod)

    return nil
}

func (mut *Mutator) Reinvoke(request *dtwebhook.ReinvocationRequest) bool {
    installPath := maputils.GetField(request.Pod.Annotations, AnnotationInstallPath, DefaultInstallPath)
    return mutateUserContainers(request.BaseRequest, installPath)
}

// ================== User Container Helpers ==================
func mutateUserContainers(request *dtwebhook.BaseRequest, installPath string) bool {
    newContainers := request.NewContainers(containerIsInjected)
    for _, container := range newContainers {
        addOneAgentToContainer(request.DynaKube, container, request.Namespace, installPath)
    }
    return len(newContainers) > 0
}

func containerIsInjected(container corev1.Container) bool {
    return mounts.IsIn(container.VolumeMounts, BinVolumeName)
}

func addOneAgentToContainer(dk dynakube.DynaKube, container *corev1.Container, namespace corev1.Namespace, installPath string) {
    logger := log.FromContext(context.Background())
    logger.Info("adding OneAgent to container", "container", container.Name)

    addVolumeMounts(container, installPath)
    addDeploymentMetadataEnv(container, dk)
    addPreloadEnv(container, installPath)
    addDtStorageEnv(container)

    if dk.Spec.NetworkZone != "" {
        addNetworkZoneEnv(container, dk.Spec.NetworkZone)
    }
    if dk.FF().IsLabelVersionDetection() {
        addVersionDetectionEnvs(container, namespace)
    }
}

// ================== Pod Annotation Helper ==================
func setInjectedAnnotation(pod *corev1.Pod) {
    if pod.Annotations == nil {
        pod.Annotations = make(map[string]string)
    }
    pod.Annotations[AnnotationInjected] = "true"
}

