package snow_test

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/etcdadm-controller/api/v1beta1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	clusterv1 "sigs.k8s.io/cluster-api/api/v1beta1"
	bootstrapv1 "sigs.k8s.io/cluster-api/bootstrap/kubeadm/api/v1beta1"
	controlplanev1 "sigs.k8s.io/cluster-api/controlplane/kubeadm/api/v1beta1"

	anywherev1 "github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/clients/kubernetes"
	"github.com/aws/eks-anywhere/pkg/constants"
	"github.com/aws/eks-anywhere/pkg/providers/snow"
	snowv1 "github.com/aws/eks-anywhere/pkg/providers/snow/api/v1beta1"
)

func TestControlPlaneObjects(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test",
			constants.EksaSystemNamespace,
			&controlplanev1.KubeadmControlPlane{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *controlplanev1.KubeadmControlPlane) error {
			obj.Spec.MachineTemplate.InfrastructureRef.Name = "test-cp-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"test-cp-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *snowv1.AWSSnowMachineTemplate) error {
			mt.DeepCopyInto(obj)
			obj.SetName("test-cp-1")
			obj.Spec.Template.Spec.InstanceType = "updated-instance-type"
			return nil
		})

	wantMachineTemplateName := "test-cp-2"
	mt.SetName(wantMachineTemplateName)
	mt.Spec.Template.Spec.InstanceType = "sbe-c.large"
	kcp := wantKubeadmControlPlane()
	kcp.Spec.MachineTemplate.InfrastructureRef.Name = wantMachineTemplateName

	got, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(Equal([]kubernetes.Object{wantCAPICluster(), kcp, wantSnowCluster(), mt, wantSnowCredentialsSecret()}))
}

func TestControlPlaneObjectsUnstackedEtcd(t *testing.T) {
	g := newSnowTest(t)
	g.clusterSpec.Cluster.Spec.ExternalEtcdConfiguration = &anywherev1.ExternalEtcdConfiguration{
		Count: 3,
		MachineGroupRef: &anywherev1.Ref{
			Kind: "SnowMachineConfig",
			Name: "test-etcd",
		},
	}
	g.clusterSpec.SnowMachineConfigs["test-etcd"] = &anywherev1.SnowMachineConfig{
		TypeMeta: metav1.TypeMeta{
			Kind: "SnowMachineConfig",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-etcd",
			Namespace: "test-namespace",
		},
		Spec: anywherev1.SnowMachineConfigSpec{
			AMIID:                    "eks-d-v1-21-5-ubuntu-ami-02833ca9a8f29c2ea",
			InstanceType:             "sbe-c.xlarge",
			SshKeyName:               "default",
			PhysicalNetworkConnector: "SFP_PLUS",
			Devices: []string{
				"1.2.3.4",
				"1.2.3.5",
			},
			OSFamily: anywherev1.Ubuntu,
			Network: snowv1.AWSSnowNetwork{
				DirectNetworkInterfaces: []snowv1.AWSSnowDirectNetworkInterface{
					{
						Index:   1,
						DHCP:    true,
						Primary: true,
					},
				},
			},
		},
	}
	mtCp := wantSnowMachineTemplate()
	mtEtcd := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test",
			constants.EksaSystemNamespace,
			&controlplanev1.KubeadmControlPlane{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *controlplanev1.KubeadmControlPlane) error {
			obj.Spec.MachineTemplate.InfrastructureRef.Name = "test-cp-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"test-cp-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *snowv1.AWSSnowMachineTemplate) error {
			mtCp.DeepCopyInto(obj)
			obj.SetName("test-cp-1")
			obj.Spec.Template.Spec.InstanceType = "updated-instance-type"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-etcd",
			constants.EksaSystemNamespace,
			&v1beta1.EtcdadmCluster{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *v1beta1.EtcdadmCluster) error {
			obj.Spec.InfrastructureTemplate.Name = "test-etcd-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"test-etcd-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *snowv1.AWSSnowMachineTemplate) error {
			mtCp.DeepCopyInto(obj)
			obj.SetName("test-etcd-1")
			obj.Spec.Template.Spec.InstanceType = "updated-instance-type"
			return nil
		})

	mtCpName := "test-cp-2"
	mtCp.SetName(mtCpName)
	mtCp.Spec.Template.Spec.InstanceType = "sbe-c.large"
	kcp := wantKubeadmControlPlaneUnstackedEtcd()
	kcp.Spec.MachineTemplate.InfrastructureRef.Name = mtCpName

	mtEtcdName := "test-etcd-2"
	mtEtcd.SetName(mtEtcdName)
	etcdCluster := wantEtcdClusterUbuntu()
	etcdCluster.Spec.InfrastructureTemplate.Name = mtEtcdName

	got, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(Equal([]kubernetes.Object{wantCAPIClusterUnstackedEtcd(), kcp, wantSnowCluster(), mtCp, etcdCluster, mtEtcd, wantSnowCredentialsSecret()}))
}

func TestControlPlaneObjectsCredentialsNil(t *testing.T) {
	g := newSnowTest(t)
	g.clusterSpec.SnowCredentialsSecret = nil
	_, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(MatchError(ContainSubstring("snowCredentialsSecret in clusterSpec shall not be nil")))
}

func TestControlPlaneObjectsSecretMissCredentialsKey(t *testing.T) {
	g := newSnowTest(t)
	g.clusterSpec.SnowCredentialsSecret.Data = map[string][]byte{
		"ca-bundle": []byte("eksa-certs"),
	}

	_, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(MatchError(ContainSubstring("unable to retrieve credentials from secret")))
}

func TestControlPlaneObjectsSecretMissCertificatesKey(t *testing.T) {
	g := newSnowTest(t)
	g.clusterSpec.SnowCredentialsSecret.Data = map[string][]byte{
		"credentials": []byte("eksa-creds"),
	}

	_, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(MatchError(ContainSubstring("unable to retrieve ca-bundle from secret")))
}

func TestControlPlaneObjectsOldControlPlaneNotExists(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test",
			constants.EksaSystemNamespace,
			&controlplanev1.KubeadmControlPlane{},
		).
		Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

	mt.SetName("snow-test-control-plane-1")
	mt.Spec.Template.Spec.InstanceType = "sbe-c.large"

	got, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(Equal([]kubernetes.Object{wantCAPICluster(), wantKubeadmControlPlane(), wantSnowCluster(), mt, wantSnowCredentialsSecret()}))
}

func TestControlPlaneObjectsOldMachineTemplateNotExists(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test",
			constants.EksaSystemNamespace,
			&controlplanev1.KubeadmControlPlane{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *controlplanev1.KubeadmControlPlane) error {
			obj.Spec.MachineTemplate.InfrastructureRef.Name = "snow-test-control-plane-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-control-plane-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

	mt.SetName("snow-test-control-plane-1")
	mt.Spec.Template.Spec.InstanceType = "sbe-c.large"

	got, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(Equal([]kubernetes.Object{wantCAPICluster(), wantKubeadmControlPlane(), wantSnowCluster(), mt, wantSnowCredentialsSecret()}))
}

func TestControlPlaneObjectsGetOldControlPlaneError(t *testing.T) {
	g := newSnowTest(t)
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test",
			constants.EksaSystemNamespace,
			&controlplanev1.KubeadmControlPlane{},
		).
		Return(errors.New("get cp error"))

	_, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).NotTo(Succeed())
}

func TestControlPlaneObjectsGetOldMachineTemplateError(t *testing.T) {
	g := newSnowTest(t)
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test",
			constants.EksaSystemNamespace,
			&controlplanev1.KubeadmControlPlane{},
		).
		Return(nil)
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		Return(errors.New("get mt error"))

	_, err := snow.ControlPlaneObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).NotTo(Succeed())
}

func TestWorkersObjects(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-1"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&bootstrapv1.KubeadmConfigTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *bootstrapv1.KubeadmConfigTemplate) error {
			wantKubeadmConfigTemplate().DeepCopyInto(obj)
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *snowv1.AWSSnowMachineTemplate) error {
			mt.DeepCopyInto(obj)
			obj.SetName("snow-test-md-0-1")
			obj.Spec.Template.Spec.InstanceType = "updated-instance-type"
			return nil
		})

	wantMachineTemplateName := "snow-test-md-0-2"
	mt.SetName(wantMachineTemplateName)
	md := wantMachineDeployment()
	md.Spec.Template.Spec.InfrastructureRef.Name = wantMachineTemplateName

	got, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(ConsistOf([]kubernetes.Object{md, wantKubeadmConfigTemplate(), mt}))
}

func TestWorkersObjectsFromBetaMachineTemplateName(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "test-wn"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "test-wn"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"test-wn",
			constants.EksaSystemNamespace,
			&bootstrapv1.KubeadmConfigTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *bootstrapv1.KubeadmConfigTemplate) error {
			wantKubeadmConfigTemplate().DeepCopyInto(obj)
			obj.SetName("test-wn")
			obj.Spec.Template.Spec.PreKubeadmCommands = []string{
				"new command",
			}
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"test-wn",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *snowv1.AWSSnowMachineTemplate) error {
			mt.DeepCopyInto(obj)
			obj.SetName("test-wn")
			obj.Spec.Template.Spec.InstanceType = "updated-instance-type"
			return nil
		})

	wantMachineTemplateName := "snow-test-md-0-1"
	mt.SetName(wantMachineTemplateName)
	md := wantMachineDeployment()
	md.Spec.Template.Spec.InfrastructureRef.Name = wantMachineTemplateName
	kct := wantKubeadmConfigTemplate()
	wantKctName := "snow-test-md-0-1"
	kct.SetName(wantKctName)
	md.Spec.Template.Spec.Bootstrap.ConfigRef.Name = wantKctName

	got, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(ConsistOf([]kubernetes.Object{md, kct, mt}))
}

func TestWorkersObjectsOldMachineDeploymentNotExists(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

	got, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(ConsistOf([]kubernetes.Object{wantMachineDeployment(), wantKubeadmConfigTemplate(), mt}))
}

func TestWorkersObjectsOldKubeadmConfigTemplateNotExists(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-1"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&bootstrapv1.KubeadmConfigTemplate{},
		).
		Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

	got, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(ConsistOf([]kubernetes.Object{wantMachineDeployment(), wantKubeadmConfigTemplate(), mt}))
}

func TestWorkersObjectsOldMachineTemplateNotExists(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-1"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&bootstrapv1.KubeadmConfigTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *bootstrapv1.KubeadmConfigTemplate) error {
			wantKubeadmConfigTemplate().DeepCopyInto(obj)
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

	got, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).To(Succeed())
	g.Expect(got).To(ConsistOf([]kubernetes.Object{wantMachineDeployment(), wantKubeadmConfigTemplate(), mt}))
}

func TestWorkersObjectsTaintsUpdated(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-1"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&bootstrapv1.KubeadmConfigTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *bootstrapv1.KubeadmConfigTemplate) error {
			wantKubeadmConfigTemplate().DeepCopyInto(obj)
			obj.Spec.Template.Spec.JoinConfiguration.NodeRegistration.Taints = []v1.Taint{
				{
					Key:    "key1",
					Value:  "val1",
					Effect: v1.TaintEffectNoExecute,
				},
			}
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *snowv1.AWSSnowMachineTemplate) error {
			mt.DeepCopyInto(obj)
			return nil
		})

	got, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)

	md := wantMachineDeployment()
	md.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-2"
	md.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-2"
	kct := wantKubeadmConfigTemplate()
	kct.SetName("snow-test-md-0-2")
	mt.SetName("snow-test-md-0-2")

	g.Expect(err).To(Succeed())
	g.Expect(got).To(ConsistOf([]kubernetes.Object{md, kct, mt}))
}

func TestWorkersObjectsLabelsUpdated(t *testing.T) {
	g := newSnowTest(t)
	mt := wantSnowMachineTemplate()
	g.clusterSpec.Cluster.Spec.WorkerNodeGroupConfigurations[0].Labels = map[string]string{
		"label1": "val1",
		"label2": "val2",
	}
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-1"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&bootstrapv1.KubeadmConfigTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *bootstrapv1.KubeadmConfigTemplate) error {
			wantKubeadmConfigTemplate().DeepCopyInto(obj)
			obj.Spec.Template.Spec.JoinConfiguration.NodeRegistration.KubeletExtraArgs = map[string]string{
				"provider-id": "aws-snow:////'{{ ds.meta_data.instance_id }}'",
				"node-labels": "label1=val2,label2=val1",
			}
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *snowv1.AWSSnowMachineTemplate) error {
			mt.DeepCopyInto(obj)
			return nil
		})

	got, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)

	md := wantMachineDeployment()
	md.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-2"
	md.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-2"
	kct := wantKubeadmConfigTemplate()
	kct.SetName("snow-test-md-0-2")
	kct.Spec.Template.Spec.JoinConfiguration.NodeRegistration.KubeletExtraArgs = map[string]string{
		"provider-id": "aws-snow:////'{{ ds.meta_data.instance_id }}'",
		"node-labels": "label1=val1,label2=val2",
	}
	mt.SetName("snow-test-md-0-2")

	g.Expect(err).To(Succeed())
	g.Expect(got).To(ContainElement(kct))
}

func TestWorkersObjectsGetMachineDeploymentError(t *testing.T) {
	g := newSnowTest(t)
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		Return(errors.New("get md error"))

	_, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).NotTo(Succeed())
}

func TestWorkersObjectsGetKubeadmConfigTemplateError(t *testing.T) {
	g := newSnowTest(t)
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-1"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&bootstrapv1.KubeadmConfigTemplate{},
		).
		Return(errors.New("get kct error"))
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		Return(nil)

	_, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).NotTo(Succeed())
}

func TestWorkersObjectsGetMachineTemplateError(t *testing.T) {
	g := newSnowTest(t)
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0",
			constants.EksaSystemNamespace,
			&clusterv1.MachineDeployment{},
		).
		DoAndReturn(func(_ context.Context, _, _ string, obj *clusterv1.MachineDeployment) error {
			wantMachineDeployment().DeepCopyInto(obj)
			obj.Spec.Template.Spec.InfrastructureRef.Name = "snow-test-md-0-1"
			obj.Spec.Template.Spec.Bootstrap.ConfigRef.Name = "snow-test-md-0-1"
			return nil
		})
	g.kubeconfigClient.EXPECT().
		Get(
			g.ctx,
			"snow-test-md-0-1",
			constants.EksaSystemNamespace,
			&snowv1.AWSSnowMachineTemplate{},
		).
		Return(errors.New("get mt error"))

	_, err := snow.WorkersObjects(g.ctx, g.logger, g.clusterSpec, g.kubeconfigClient)
	g.Expect(err).NotTo(Succeed())
}

func TestKubeadmConfigTemplatesWithRegistryMirror(t *testing.T) {
	for _, tt := range registryMirrorTests {
		t.Run(tt.name, func(t *testing.T) {
			g := newSnowTest(t)

			g.kubeconfigClient.EXPECT().
				Get(
					g.ctx,
					"snow-test-md-0",
					constants.EksaSystemNamespace,
					&clusterv1.MachineDeployment{},
				).
				Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

			g.clusterSpec.Cluster.Spec.RegistryMirrorConfiguration = tt.registryMirrorConfig
			gotMt, gotKct, err := snow.WorkersMachineAndConfigTemplate(g.ctx, g.logger, g.kubeconfigClient, g.clusterSpec)
			g.Expect(err).To(Succeed())
			wantMt := map[string]*snowv1.AWSSnowMachineTemplate{
				"md-0": wantSnowMachineTemplate(),
			}
			wantKct := map[string]*bootstrapv1.KubeadmConfigTemplate{
				"md-0": wantKubeadmConfigTemplate(),
			}
			wantKct["md-0"].Spec.Template.Spec.Files = tt.wantFiles
			wantKct["md-0"].Spec.Template.Spec.PreKubeadmCommands = append(wantKct["md-0"].Spec.Template.Spec.PreKubeadmCommands, wantRegistryMirrorCommands()...)
			g.Expect(gotMt).To(Equal(wantMt))
			g.Expect(gotKct).To(Equal(wantKct))
		})
	}
}

func TestKubeadmConfigTemplatesWithProxyConfig(t *testing.T) {
	for _, tt := range proxyTests {
		t.Run(tt.name, func(t *testing.T) {
			g := newSnowTest(t)

			g.kubeconfigClient.EXPECT().
				Get(
					g.ctx,
					"snow-test-md-0",
					constants.EksaSystemNamespace,
					&clusterv1.MachineDeployment{},
				).
				Return(apierrors.NewNotFound(schema.GroupResource{Group: "", Resource: ""}, ""))

			g.clusterSpec.Cluster.Spec.ProxyConfiguration = tt.proxy

			_, got, err := snow.WorkersMachineAndConfigTemplate(g.ctx, g.logger, g.kubeconfigClient, g.clusterSpec)
			g.Expect(err).To(Succeed())
			want := map[string]*bootstrapv1.KubeadmConfigTemplate{
				"md-0": wantKubeadmConfigTemplate(),
			}
			want["md-0"].Spec.Template.Spec.Files = tt.wantFiles
			want["md-0"].Spec.Template.Spec.PreKubeadmCommands = append(want["md-0"].Spec.Template.Spec.PreKubeadmCommands, wantProxyConfigCommands()...)
			g.Expect(got).To(Equal(want))
		})
	}
}

func TestNewMachineTemplateName(t *testing.T) {
	tests := []struct {
		name     string
		old, new *snowv1.AWSSnowMachineTemplate
		want     string
	}{
		{
			name: "remove one device",
			old: &snowv1.AWSSnowMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "old-1",
				},
				Spec: snowv1.AWSSnowMachineTemplateSpec{
					Template: snowv1.AWSSnowMachineTemplateResource{
						Spec: snowv1.AWSSnowMachineSpec{
							Devices: []string{
								"1",
								"2",
							},
						},
					},
				},
			},
			new: &snowv1.AWSSnowMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-1",
				},
				Spec: snowv1.AWSSnowMachineTemplateSpec{
					Template: snowv1.AWSSnowMachineTemplateResource{
						Spec: snowv1.AWSSnowMachineSpec{
							Devices: []string{
								"1",
							},
						},
					},
				},
			},
			want: "old-2",
		},
		{
			name: "add one device",
			old: &snowv1.AWSSnowMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "old-1",
				},
				Spec: snowv1.AWSSnowMachineTemplateSpec{
					Template: snowv1.AWSSnowMachineTemplateResource{
						Spec: snowv1.AWSSnowMachineSpec{
							Devices: []string{
								"1",
							},
						},
					},
				},
			},
			new: &snowv1.AWSSnowMachineTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name: "new-1",
				},
				Spec: snowv1.AWSSnowMachineTemplateSpec{
					Template: snowv1.AWSSnowMachineTemplateResource{
						Spec: snowv1.AWSSnowMachineSpec{
							Devices: []string{
								"1",
								"2",
							},
						},
					},
				},
			},
			want: "old-2",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := newSnowTest(t)
			name := snow.NewMachineTemplateName(tt.new, tt.old)
			g.Expect(name).To(Equal(tt.want))
		})
	}
}
