package v1alpha1_test

import (
	"testing"

	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	snowv1 "github.com/aws/eks-anywhere/pkg/providers/snow/api/v1beta1"
)

func TestSnowMachineConfigSetDefaults(t *testing.T) {
	g := NewWithT(t)

	sOld := snowMachineConfig()
	sOld.Default()

	g.Expect(sOld.Spec.InstanceType).To(Equal(v1alpha1.DefaultSnowInstanceType))
	g.Expect(sOld.Spec.PhysicalNetworkConnector).To(Equal(v1alpha1.DefaultSnowPhysicalNetworkConnectorType))
}

func TestSnowMachineConfigValidateCreateNoAMI(t *testing.T) {
	g := NewWithT(t)

	sOld := snowMachineConfig()
	sOld.Spec.InstanceType = v1alpha1.SbeCLarge
	sOld.Spec.Devices = []string{"1.2.3.4"}
	sOld.Spec.OSFamily = v1alpha1.Bottlerocket
	sOld.Spec.ContainersVolume = &snowv1.Volume{
		Size: 25,
	}
	sOld.Spec.Network = snowv1.AWSSnowNetwork{
		DirectNetworkInterfaces: []snowv1.AWSSnowDirectNetworkInterface{
			{
				Index:   1,
				DHCP:    true,
				Primary: true,
			},
		},
	}

	g.Expect(sOld.ValidateCreate()).To(Succeed())
}

func TestSnowMachineConfigValidateCreateInvalidInstanceType(t *testing.T) {
	g := NewWithT(t)

	sOld := snowMachineConfig()
	sOld.Spec.InstanceType = "invalid-instance-type"

	g.Expect(sOld.ValidateCreate()).To(MatchError(ContainSubstring("SnowMachineConfig InstanceType invalid-instance-type is not supported")))
}

func TestSnowMachineConfigValidateCreate(t *testing.T) {
	g := NewWithT(t)

	sOld := snowMachineConfig()
	sOld.Spec.AMIID = "testAMI"
	sOld.Spec.InstanceType = v1alpha1.SbeCLarge
	sOld.Spec.Devices = []string{"1.2.3.4"}
	sOld.Spec.OSFamily = v1alpha1.Bottlerocket
	sOld.Spec.ContainersVolume = &snowv1.Volume{
		Size: 25,
	}
	sOld.Spec.Network = snowv1.AWSSnowNetwork{
		DirectNetworkInterfaces: []snowv1.AWSSnowDirectNetworkInterface{
			{
				Index:   1,
				DHCP:    true,
				Primary: true,
			},
		},
	}

	g.Expect(sOld.ValidateCreate()).To(Succeed())
}

func TestSnowMachineConfigValidateUpdate(t *testing.T) {
	g := NewWithT(t)

	sOld := snowMachineConfig()
	sNew := sOld.DeepCopy()
	sNew.Spec.AMIID = "testAMI"
	sNew.Spec.InstanceType = v1alpha1.SbeCLarge
	sNew.Spec.Devices = []string{"1.2.3.4"}
	sNew.Spec.OSFamily = v1alpha1.Bottlerocket
	sNew.Spec.ContainersVolume = &snowv1.Volume{
		Size: 25,
	}
	sNew.Spec.Network = snowv1.AWSSnowNetwork{
		DirectNetworkInterfaces: []snowv1.AWSSnowDirectNetworkInterface{
			{
				Index:   1,
				DHCP:    true,
				Primary: true,
			},
		},
	}

	g.Expect(sNew.ValidateUpdate(&sOld)).To(Succeed())
}

func TestSnowMachineConfigValidateUpdateNoDevices(t *testing.T) {
	g := NewWithT(t)

	sOld := snowMachineConfig()
	sNew := sOld.DeepCopy()
	sNew.Spec.AMIID = "testAMI"
	sNew.Spec.InstanceType = v1alpha1.SbeCLarge
	sNew.Spec.OSFamily = v1alpha1.Bottlerocket

	g.Expect(sNew.ValidateUpdate(&sOld)).To(MatchError(ContainSubstring("Devices must contain at least one device IP")))
}

// Unit test to pass the code coverage job.
func TestSnowMachineConfigValidateDelete(t *testing.T) {
	g := NewWithT(t)
	sOld := snowMachineConfig()
	g.Expect(sOld.ValidateDelete()).To(Succeed())
}

func snowMachineConfig() v1alpha1.SnowMachineConfig {
	return v1alpha1.SnowMachineConfig{
		TypeMeta:   metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{Annotations: make(map[string]string, 2)},
		Spec:       v1alpha1.SnowMachineConfigSpec{},
		Status:     v1alpha1.SnowMachineConfigStatus{},
	}
}
