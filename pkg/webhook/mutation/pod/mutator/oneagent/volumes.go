package oneagent

import (
	"context"
	"github.com/rkitindi-kr/dynatrace-bootstrapper/pkg/configure/oneagent/preload"
	"github.com/rkitindi-kr/dynatrace-operator/pkg/consts"
	dtcsi "github.com/rkitindi-kr/dynatrace-operator/pkg/controllers/csi"
	csivolumes "github.com/rkitindi-kr/dynatrace-operator/pkg/controllers/csi/driver/volumes"
	appvolumes "github.com/rkitindi-kr/dynatrace-operator/pkg/controllers/csi/driver/volumes/app"
	volumeutils "github.com/rkitindi-kr/dynatrace-operator/pkg/util/kubeobjects/volumes"
	"github.com/rkitindi-kr/dynatrace-operator/pkg/webhook/mutation/pod/volumes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"fmt"
        "k8s.io/apimachinery/pkg/api/resource"
        metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
        "sigs.k8s.io/controller-runtime/pkg/client"

        volumeutils "github.com/rkitindi-kr/dynatrace-operator/pkg/util/kubeobjects/volumes"
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



// Above funtion is replaced by this function below which introduced pod pending problem:

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

        if _, ok := pod.Annotations["pvc-webhook/storage-size"]; !ok {
        pod.Annotations["pvc-webhook/storage-size"] = "2Gi"
    }
    if _, ok := pod.Annotations["pvc-webhook/storage-class"]; !ok {
        pod.Annotations["pvc-webhook/storage-class"] = "robin-repl-3"
    }
    if _, ok := pod.Annotations["pvc-webhook/claim"]; !ok {
        pod.Annotations["pvc-webhook/claim"] = pvcName
    }

}


// Above function has been replaced by function below which adds PVC creation capability needed to solve POD pending problem


*/

// addPVCBinVolume injects a PVC-backed volume into the Pod AND ensures the PVC exists
func addPVCBinVolume(ctx context.Context, c client.Client, pod *corev1.Pod, defaultSize, defaultClass string) error {
    if volumeutils.IsIn(pod.Spec.Volumes, BinVolumeName) {
        return nil
    }

    // build deterministic PVC name (namespace + pod + volume)
    pvcName := fmt.Sprintf("%s-%s", BinVolumeName, pod.Name)

    // attach volume to Pod
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

    // ensure annotations exist (optional, but good for traceability)
    if pod.Annotations == nil {
        pod.Annotations = map[string]string{}
    }
    if _, ok := pod.Annotations["pvc-webhook/storage-size"]; !ok {
        pod.Annotations["pvc-webhook/storage-size"] = "2Gi"
    }
    if _, ok := pod.Annotations["pvc-webhook/storage-class"]; !ok && defaultClass != "" {
        pod.Annotations["pvc-webhook/storage-class"] = "robin-repl-3"
    }
    if _, ok := pod.Annotations["pvc-webhook/claim"]; !ok {
        pod.Annotations["pvc-webhook/claim"] = pvcName
    }

    // --- NEW: ensure PVC exists ---
    var pvc corev1.PersistentVolumeClaim
    err := c.Get(ctx, client.ObjectKey{Namespace: pod.Namespace, Name: pvcName}, &pvc)
    if err == nil {
        return nil // already exists
    }

    // Create PVC
    storageSize := pod.Annotations["pvc-webhook/storage-size"]
    sc := pod.Annotations["pvc-webhook/storage-class"]

    pvc = corev1.PersistentVolumeClaim{
        ObjectMeta: metav1.ObjectMeta{
            Name:      pvcName,
            Namespace: pod.Namespace,
            Labels: map[string]string{
                "created-by": "pvc-webhook",
                "pod":        pod.Name,
            },
        },
        Spec: corev1.PersistentVolumeClaimSpec{
            AccessModes: []corev1.PersistentVolumeAccessMode{
                corev1.ReadWriteOnce,
            },
            Resources: corev1.VolumeResourceRequirements{
                Requests: corev1.ResourceList{
                    corev1.ResourceStorage: resource.MustParse(storageSize),
                },
            },
        },
    }

    if sc != "" {
        pvc.Spec.StorageClassName = &sc
    }

    if err := c.Create(ctx, &pvc); err != nil {
        return fmt.Errorf("failed to create pvc %s: %w", pvcName, err)
    }

    return nil
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
