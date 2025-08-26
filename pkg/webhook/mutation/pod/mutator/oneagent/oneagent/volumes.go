package oneagent

import (
	"github.com/rkitindi-kr/dynatrace-bootstrapper/pkg/configure/oneagent/preload"
	"github.com/rkitindi-kr/dynatrace-operator/pkg/consts"
	dtcsi "github.com/rkitindi-kr/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/rkitindi-kr/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	appvolumes "github.com/rkitindi-kr/dynatrace-operator/pkg/controllers/csi/driver/volumes/app"
	volumeutils "github.com/rkitindi-kr/dynatrace-operator/pkg/util/kubeobjects/volumes"
	"github.com/rkitindi-kr/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/utils/ptr"
)

const (
	BinVolumeName    = "oneagent-bin"
	ldPreloadPath    = "/etc/ld.so.preload"
	ldPreloadSubPath = preload.ConfigPath
)

func addVolumeMounts(container *corev1.Container, installPath string) {
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      BinVolumeName,
			MountPath: installPath,
			ReadOnly:  true,
		},
		corev1.VolumeMount{
			Name:      volumes.ConfigVolumeName,
			MountPath: ldPreloadPath,
			SubPath:   ldPreloadSubPath,
		},
	)
}

func addInitBinMount(initContainer *corev1.Container, readonly bool) {
	initContainer.VolumeMounts = append(initContainer.VolumeMounts,
		corev1.VolumeMount{
			Name:      BinVolumeName,
			MountPath: consts.AgentInitBinDirMount,
			ReadOnly:  readonly,
		},
	)
}

/*
func addEmptyDirBinVolume(pod *corev1.Pod) {
	if volumeutils.IsIn(pod.Spec.Volumes, BinVolumeName) {
		return
	}

	emptyDirVS := corev1.EmptyDirVolumeSource{}

	if r, ok := pod.Annotations[AnnotationOneAgenBinResource]; ok && r != "" {
		sizeLimit, err := resource.ParseQuantity(r)
		if err != nil {
			log.Error(err, "failed to parse quantity from annotation "+AnnotationOneAgenBinResource, "value", r)
		} else {
			emptyDirVS = corev1.EmptyDirVolumeSource{
				SizeLimit: &sizeLimit,
			}
		}
	}

	volumeSource := corev1.VolumeSource{
		EmptyDir: &emptyDirVS,
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name:         BinVolumeName,
			VolumeSource: volumeSource,
		},
	)
}

*/

// Above funtion is replaced by this:

func addPVCBinVolume(pod *corev1.Pod, defaultSize, defaultClass string) {
    if volumeutils.IsIn(pod.Spec.Volumes, BinVolumeName) {
        return
    }

    // build deterministic PVC name (namespace + pod + volume)
    pvcName := fmt.Sprintf("%s-%s", BinVolumeName, pod.Name)

    volumeSource := corev1.VolumeSource{
        PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
            ClaimName: pvcName,
        },
    }

    pod.Spec.Volumes = append(pod.Spec.Volumes,
        corev1.Volume{
            Name:         BinVolumeName,
            VolumeSource: volumeSource,
        },
    )

    // annotate pod with PVC metadata so controller can pick it up
    if pod.Annotations == nil {
        pod.Annotations = map[string]string{}
    }
    pod.Annotations["pvc-webhook/storage-size"] = defaultSize
    pod.Annotations["pvc-webhook/storage-class"] = defaultClass
    pod.Annotations["pvc-webhook/claim"] = pvcName
}

// The end of new function

func addCSIBinVolume(pod *corev1.Pod, dkName string, maxTimeout string) {
	if volumeutils.IsIn(pod.Spec.Volumes, BinVolumeName) {
		return
	}

	volumeSource := corev1.VolumeSource{
		CSI: &corev1.CSIVolumeSource{
			Driver:   dtcsi.DriverName,
			ReadOnly: ptr.To(true),
			VolumeAttributes: map[string]string{
				csivolumes.CSIVolumeAttributeModeField:     appvolumes.Mode,
				csivolumes.CSIVolumeAttributeDynakubeField: dkName,
				csivolumes.CSIVolumeAttributeRetryTimeout:  maxTimeout,
			},
		},
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes,
		corev1.Volume{
			Name:         BinVolumeName,
			VolumeSource: volumeSource,
		},
	)
}
