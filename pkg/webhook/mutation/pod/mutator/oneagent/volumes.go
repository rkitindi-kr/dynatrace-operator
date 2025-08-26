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
	AnnotationOneAgenBinResource = "pvc-webhook/storage-size"   // or your prior key for size; keep existing if different
        AnnotationStorageClass       = "pvc-webhook/storage-class" // optional
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

// addEphemeralBinVolume converts the previous emptyDir volume into a Generic Ephemeral Volume
// backed by a PVC created via VolumeClaimTemplate. Kubernetes will create/bind the PVC before
// starting containers and delete it when the Pod is deleted (no external controller needed).
func addEphemeralBinVolume(pod *corev1.Pod) {
	if pod == nil {
		return
	}

	// --- derive requested size ---
	sizeStr := "2Gi"
	if r, ok := pod.Annotations[AnnotationOneAgenBinResource]; ok && r != "" {
		sizeStr = r
	}
	qty, err := resource.ParseQuantity(sizeStr)
	if err != nil {
		// fall back safely
		qty = resource.MustParse("2Gi")
	}

	// --- optional storage class ---
	var scPtr *string
	if sc, ok := pod.Annotations[AnnotationStorageClass]; ok && sc != "" {
		scPtr = &sc
	}

	// --- build the ephemeral volume ---
	vol := corev1.Volume{
		Name: BinVolumeName, // keep your existing constant
		VolumeSource: corev1.VolumeSource{
			Ephemeral: &corev1.EphemeralVolumeSource{
				VolumeClaimTemplate: &corev1.PersistentVolumeClaimTemplate{
					ObjectMeta: metav1.ObjectMeta{
						// Add labels/annotations if you want them propagated to the PVC
						Labels: map[string]string{
							"created-by": "oneagent-mutator",
							"pod":        pod.Name,
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{
						AccessModes: []corev1.PersistentVolumeAccessMode{
							corev1.ReadWriteOnce, // change if you need RWX
						},
						Resources: corev1.VolumeResourceRequirements{
							Requests: corev1.ResourceList{
								corev1.ResourceStorage: qty,
							},
						},
						StorageClassName: scPtr, // nil => default storage class
					},
				},
			},
		},
	}

	// --- replace existing volume with same name, or append if missing ---
	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == BinVolumeName {
			pod.Spec.Volumes[i] = vol
			return
		}
	}
	pod.Spec.Volumes = append(pod.Spec.Volumes, vol)
}


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
