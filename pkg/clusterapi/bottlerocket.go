package clusterapi

import (
	etcdbootstrapv1 "github.com/aws/etcdadm-bootstrap-provider/api/v1beta1"
	etcdv1 "github.com/aws/etcdadm-controller/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	"github.com/aws/eks-anywhere/pkg/cluster"
	"github.com/aws/eks-anywhere/release/api/v1alpha1"
)

func bottlerocketBootstrap(image v1alpha1.Image) bootstrapv1.BottlerocketBootstrap {
	return bootstrapv1.BottlerocketBootstrap{
		ImageMeta: bootstrapv1.ImageMeta{
			ImageRepository: image.Image(),
			ImageTag:        image.Tag(),
		},
	}
}

func bottlerocketAdmin(image v1alpha1.Image) bootstrapv1.BottlerocketAdmin {
	return bootstrapv1.BottlerocketAdmin{
		ImageMeta: bootstrapv1.ImageMeta{
			ImageRepository: image.Image(),
			ImageTag:        image.Tag(),
		},
	}
}

func bottlerocketControl(image v1alpha1.Image) bootstrapv1.BottlerocketControl {
	return bootstrapv1.BottlerocketControl{
		ImageMeta: bootstrapv1.ImageMeta{
			ImageRepository: image.Image(),
			ImageTag:        image.Tag(),
		},
	}
}

func pause(image v1alpha1.Image) bootstrapv1.Pause {
	return bootstrapv1.Pause{
		ImageMeta: bootstrapv1.ImageMeta{
			ImageRepository: image.Image(),
			ImageTag:        image.Tag(),
		},
	}
}

// SetBottlerocketInKubeadmControlPlane adds bottlerocket bootstrap image metadata in kubeadmControlPlane.
func SetBottlerocketInKubeadmControlPlane(kcp *controlplanev1.KubeadmControlPlane, versionsBundle *cluster.VersionsBundle) {
	b := bottlerocketBootstrap(versionsBundle.BottleRocketHostContainers.KubeadmBootstrap)
	p := pause(versionsBundle.KubeDistro.Pause)
	kcp.Spec.KubeadmConfigSpec.Format = bootstrapv1.Bottlerocket
	kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.BottlerocketBootstrap = b
	kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.Pause = p
	kcp.Spec.KubeadmConfigSpec.JoinConfiguration.BottlerocketBootstrap = b
	kcp.Spec.KubeadmConfigSpec.JoinConfiguration.Pause = p
}

// SetBottlerocketAdminContainerImageInKubeadmControlPlane overrides the default bottlerocket admin container image metadata in kubeadmControlPlane.
func SetBottlerocketAdminContainerImageInKubeadmControlPlane(kcp *controlplanev1.KubeadmControlPlane, versionsBundle *cluster.VersionsBundle) {
	b := bottlerocketAdmin(versionsBundle.BottleRocketHostContainers.Admin)
	kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.BottlerocketAdmin = b
	kcp.Spec.KubeadmConfigSpec.JoinConfiguration.BottlerocketAdmin = b
}

// SetBottlerocketControlContainerImageInKubeadmControlPlane overrides the default bottlerocket control container image metadata in kubeadmControlPlane.
func SetBottlerocketControlContainerImageInKubeadmControlPlane(kcp *controlplanev1.KubeadmControlPlane, versionsBundle *cluster.VersionsBundle) {
	b := bottlerocketControl(versionsBundle.BottleRocketHostContainers.Control)
	kcp.Spec.KubeadmConfigSpec.ClusterConfiguration.BottlerocketControl = b
	kcp.Spec.KubeadmConfigSpec.JoinConfiguration.BottlerocketControl = b
}

// SetBottlerocketInKubeadmConfigTemplate adds bottlerocket bootstrap image metadata in kubeadmConfigTemplate.
func SetBottlerocketInKubeadmConfigTemplate(kct *bootstrapv1.KubeadmConfigTemplate, versionsBundle *cluster.VersionsBundle) {
	kct.Spec.Template.Spec.Format = bootstrapv1.Bottlerocket
	kct.Spec.Template.Spec.JoinConfiguration.BottlerocketBootstrap = bottlerocketBootstrap(versionsBundle.BottleRocketHostContainers.KubeadmBootstrap)
	kct.Spec.Template.Spec.JoinConfiguration.Pause = pause(versionsBundle.KubeDistro.Pause)
}

// SetBottlerocketAdminContainerImageInKubeadmConfigTemplate overrides the default bottlerocket admin container image metadata in kubeadmConfigTemplate.
func SetBottlerocketAdminContainerImageInKubeadmConfigTemplate(kct *bootstrapv1.KubeadmConfigTemplate, versionsBundle *cluster.VersionsBundle) {
	kct.Spec.Template.Spec.JoinConfiguration.BottlerocketAdmin = bottlerocketAdmin(versionsBundle.BottleRocketHostContainers.Admin)
}

// SetBottlerocketControlContainerImageInKubeadmConfigTemplate overrides the default bottlerocket control container image metadata in kubeadmConfigTemplate.
func SetBottlerocketControlContainerImageInKubeadmConfigTemplate(kct *bootstrapv1.KubeadmConfigTemplate, versionsBundle *cluster.VersionsBundle) {
	kct.Spec.Template.Spec.JoinConfiguration.BottlerocketControl = bottlerocketControl(versionsBundle.BottleRocketHostContainers.Control)
}

// SetBottlerocketInEtcdCluster adds bottlerocket config in etcdadmCluster.
func SetBottlerocketInEtcdCluster(etcd *etcdv1.EtcdadmCluster, versionsBundle *cluster.VersionsBundle) {
	etcd.Spec.EtcdadmConfigSpec.BottlerocketConfig = &etcdbootstrapv1.BottlerocketConfig{
		EtcdImage:      versionsBundle.KubeDistro.EtcdImage.VersionedImage(),
		BootstrapImage: versionsBundle.BottleRocketHostContainers.KubeadmBootstrap.VersionedImage(),
		PauseImage:     versionsBundle.KubeDistro.Pause.VersionedImage(),
	}
}

// SetBottlerocketAdminContainerImageInEtcdCluster overrides the default bottlerocket admin container image metadata in etcdadmCluster.
func SetBottlerocketAdminContainerImageInEtcdCluster(etcd *etcdv1.EtcdadmCluster, adminImage v1alpha1.Image) {
	etcd.Spec.EtcdadmConfigSpec.BottlerocketConfig.AdminImage = adminImage.VersionedImage()
}

// SetBottlerocketControlContainerImageInEtcdCluster overrides the default bottlerocket control container image metadata in etcdadmCluster.
func SetBottlerocketControlContainerImageInEtcdCluster(etcd *etcdv1.EtcdadmCluster, controlImage v1alpha1.Image) {
	etcd.Spec.EtcdadmConfigSpec.BottlerocketConfig.ControlImage = controlImage.VersionedImage()
}
