//go:build e2e
// +build e2e

package e2e

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/eks-anywhere/internal/pkg/api"
	"github.com/aws/eks-anywhere/pkg/api/v1alpha1"
	"github.com/aws/eks-anywhere/pkg/types"
	"github.com/aws/eks-anywhere/test/framework"
)

func runWorkloadClusterFlow(test *framework.MulticlusterE2ETest) {
	test.CreateManagementCluster()
	test.RunInWorkloadClusters(func(w *framework.WorkloadCluster) {
		w.GenerateClusterConfig()
		w.CreateCluster()
		w.DeleteCluster()
	})
	time.Sleep(5 * time.Minute)
	test.DeleteManagementCluster()
}

func runTinkerbellWorkloadClusterFlow(test *framework.MulticlusterE2ETest) {
	test.CreateTinkerbellManagementCluster()
	test.RunInWorkloadClusters(func(w *framework.WorkloadCluster) {
		w.GenerateClusterConfig()
		w.PowerOffHardware()
		w.CreateCluster(framework.WithForce(), framework.WithControlPlaneWaitTimeout("20m"))
		w.StopIfFailed()
		w.DeleteCluster()
		w.ValidateHardwareDecommissioned()
	})
	test.DeleteTinkerbellManagementCluster()
}

func runSimpleWorkloadUpgradeFlowForBareMetal(test *framework.MulticlusterE2ETest, updateVersion v1alpha1.KubernetesVersion, clusterOpts ...framework.ClusterE2ETestOpt) {
	test.CreateTinkerbellManagementCluster()
	test.RunInWorkloadClusters(func(w *framework.WorkloadCluster) {
		w.GenerateClusterConfig()
		w.PowerOffHardware()
		w.CreateCluster(framework.WithForce(), framework.WithControlPlaneWaitTimeout("20m"))
		time.Sleep(2 * time.Minute)
		w.UpgradeCluster(clusterOpts)
		time.Sleep(2 * time.Minute)
		w.ValidateCluster(updateVersion)
		w.StopIfFailed()
		w.DeleteCluster()
		w.ValidateHardwareDecommissioned()
	})
	test.DeleteManagementCluster()
}

func runTinkerbellWorkloadClusterFlowSkipPowerActions(test *framework.MulticlusterE2ETest) {
	test.CreateTinkerbellManagementCluster()
	test.RunInWorkloadClusters(func(w *framework.WorkloadCluster) {
		w.GenerateClusterConfig()
		w.PowerOffHardware()
		w.PXEBootHardware()
		w.PowerOnHardware()
		w.CreateCluster(framework.WithForce(), framework.WithControlPlaneWaitTimeout("20m"))
		w.StopIfFailed()
		w.DeleteCluster()
		w.PowerOffHardware()
		w.ValidateHardwareDecommissioned()
	})
	test.ManagementCluster.StopIfFailed()
	test.ManagementCluster.DeleteCluster()
	test.ManagementCluster.PowerOffHardware()
	test.ManagementCluster.ValidateHardwareDecommissioned()
}

func runWorkloadClusterFlowWithGitOps(test *framework.MulticlusterE2ETest, clusterOpts ...framework.ClusterE2ETestOpt) {
	test.CreateManagementCluster()
	test.RunInWorkloadClusters(func(w *framework.WorkloadCluster) {
		w.GenerateClusterConfig()
		w.CreateCluster()
		w.UpgradeWithGitOps(clusterOpts...)
		time.Sleep(5 * time.Minute)
		w.DeleteCluster()
	})
	time.Sleep(5 * time.Minute)
	test.DeleteManagementCluster()
}

func runWorkloadClusterUpgradeFlowCheckWorkloadRollingUpgrade(test *framework.MulticlusterE2ETest, clusterOpts ...framework.ClusterE2ETestOpt) {
	latest := latestMinorRelease(test.T)
	test.CreateManagementClusterForVersion(latest.Version, framework.ExecuteWithEksaRelease(latest))
	test.RunInWorkloadClusters(func(w *framework.WorkloadCluster) {
		w.GenerateClusterConfigForVersion(latest.Version)
		w.CreateCluster(framework.ExecuteWithEksaRelease(latest))
	})
	preUpgradeMachines := make(map[string]map[string]types.Machine, 0)
	for key, workloadCluster := range test.WorkloadClusters {
		test.T.Logf("Capturing CAPI machines for cluster %v", workloadCluster)
		mdName := fmt.Sprintf("%s-%s", workloadCluster.ClusterName, "md-0")
		test.ManagementCluster.WaitForMachineDeploymentReady(mdName)
		preUpgradeMachines[key] = test.ManagementCluster.GetCapiMachinesForCluster(workloadCluster.ClusterName)
	}
	test.ManagementCluster.UpgradeCluster(clusterOpts)
	test.T.Logf("Waiting for EKS-A controller to reconcile clusters")
	time.Sleep(2 * time.Minute) // Time for new eks-a controller to kick in and potentially trigger rolling upgrade
	for key, workloadCluster := range test.WorkloadClusters {
		test.T.Logf("Capturing CAPI machines for cluster %v", workloadCluster)
		postUpgradeMachines := test.ManagementCluster.GetCapiMachinesForCluster(workloadCluster.ClusterName)
		if anyMachinesChanged(preUpgradeMachines[key], postUpgradeMachines) {
			test.T.Fatalf("Found CAPI machines of workload cluster were changed after upgrading management cluster")
		}
	}
	test.RunInWorkloadClusters(func(w *framework.WorkloadCluster) {
		w.DeleteCluster()
	})
	test.DeleteManagementCluster()
}

func anyMachinesChanged(machineMap1 map[string]types.Machine, machineMap2 map[string]types.Machine) bool {
	if len(machineMap1) != len(machineMap2) {
		return true
	}
	for machineName := range machineMap1 {
		if _, found := machineMap2[machineName]; !found {
			return true
		}
	}
	return false
}

func TestVSphereKubernetes121MulticlusterWorkloadCluster(t *testing.T) {
	provider := framework.NewVSphere(t, framework.WithUbuntu121())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
	)
	runWorkloadClusterFlow(test)
}

func TestVSphereUpgradeMulticlusterWorkloadClusterWithFluxLegacy(t *testing.T) {
	provider := framework.NewVSphere(t, framework.WithUbuntu121())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxLegacy(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxLegacy(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
	)
	runWorkloadClusterFlowWithGitOps(
		test,
		framework.WithClusterUpgradeGit(
			api.WithKubernetesVersion(v1alpha1.Kube122),
			api.WithControlPlaneCount(3),
			api.WithWorkerNodeCount(3),
		),
		provider.WithProviderUpgradeGit(
			provider.Ubuntu122Template(),
		),
	)
}

func TestDockerUpgradeWorkloadClusterWithFluxLegacy(t *testing.T) {
	provider := framework.NewDocker(t)
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxLegacy(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxLegacy(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
			),
		),
	)
	runWorkloadClusterFlowWithGitOps(
		test,
		framework.WithClusterUpgradeGit(
			api.WithKubernetesVersion(v1alpha1.Kube122),
			api.WithControlPlaneCount(2),
			api.WithWorkerNodeCount(2),
		),
		// Needed in order to replace the DockerDatacenterConfig namespace field with the value specified
		// compared to when it was initially created without it.
		provider.WithProviderUpgradeGit(),
	)
}

func TestDockerUpgradeWorkloadClusterWithGithubFlux(t *testing.T) {
	provider := framework.NewDocker(t)
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxGithub(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxGithub(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
			),
		),
	)
	runWorkloadClusterFlowWithGitOps(
		test,
		framework.WithClusterUpgradeGit(
			api.WithKubernetesVersion(v1alpha1.Kube122),
			api.WithControlPlaneCount(2),
			api.WithWorkerNodeCount(2),
		),
		// Needed in order to replace the DockerDatacenterConfig namespace field with the value specified
		// compared to when it was initially created without it.
		provider.WithProviderUpgradeGit(),
	)
}

func TestVSphereUpgradeMulticlusterWorkloadClusterWithGithubFlux(t *testing.T) {
	provider := framework.NewVSphere(t, framework.WithUbuntu121())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxGithub(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxGithub(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
	)
	runWorkloadClusterFlowWithGitOps(
		test,
		framework.WithClusterUpgradeGit(
			api.WithKubernetesVersion(v1alpha1.Kube122),
			api.WithControlPlaneCount(3),
			api.WithWorkerNodeCount(3),
		),
		provider.WithProviderUpgradeGit(
			provider.Ubuntu122Template(),
		),
	)
}

func TestCloudStackKubernetes121WorkloadCluster(t *testing.T) {
	provider := framework.NewCloudStack(t, framework.WithCloudStackRedhat121())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
	)
	runWorkloadClusterFlow(test)
}

func TestCloudStackUpgradeMulticlusterWorkloadClusterWithFluxLegacy(t *testing.T) {
	provider := framework.NewCloudStack(t, framework.WithCloudStackRedhat122())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxLegacy(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube122),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithFluxLegacy(),
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube122),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithStackedEtcdTopology(),
			),
		),
	)
	runWorkloadClusterFlowWithGitOps(
		test,
		framework.WithClusterUpgradeGit(
			api.WithKubernetesVersion(v1alpha1.Kube123),
			api.WithControlPlaneCount(3),
			api.WithWorkerNodeCount(3),
		),
		provider.WithProviderUpgradeGit(
			framework.UpdateRedhatTemplate123Var(),
		),
	)
}

func TestCloudStackKubernetes121ManagementClusterUpgradeFromLatest(t *testing.T) {
	provider := framework.NewCloudStack(t, framework.WithCloudStackRedhat121())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithEtcdCountIfExternal(1),
			),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube121),
				api.WithControlPlaneCount(1),
				api.WithWorkerNodeCount(1),
				api.WithEtcdCountIfExternal(1),
			),
		),
	)
	runWorkloadClusterUpgradeFlowCheckWorkloadRollingUpgrade(test)
}

func TestTinkerbellKubernetes122UbuntuWorkloadCluster(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithUbuntu122Tinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithControlPlaneHardware(2),
			framework.WithWorkerHardware(2),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
		),
	)
	runTinkerbellWorkloadClusterFlow(test)
}

func TestTinkerbellKubernetes122BottlerocketWorkloadCluster(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithBottleRocketTinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithControlPlaneHardware(2),
			framework.WithWorkerHardware(2),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
		),
	)
	runTinkerbellWorkloadClusterFlow(test)
}

func TestTinkerbellKubernetes122UbuntuSingleNodeWorkloadCluster(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithUbuntu122Tinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube122),
				api.WithEtcdCountIfExternal(0),
				api.RemoveAllWorkerNodeGroups(),
			),
			framework.WithControlPlaneHardware(2),
			framework.WithWorkerHardware(0),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube122),
				api.WithEtcdCountIfExternal(0),
				api.RemoveAllWorkerNodeGroups(),
			),
		),
	)
	runTinkerbellWorkloadClusterFlow(test)
}

func TestTinkerbellKubernetes122BottlerocketSingleNodeWorkloadCluster(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithBottleRocketTinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube122),
				api.WithEtcdCountIfExternal(0),
				api.RemoveAllWorkerNodeGroups(),
			),
			framework.WithControlPlaneHardware(2),
			framework.WithWorkerHardware(0),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(
				api.WithKubernetesVersion(v1alpha1.Kube122),
				api.WithEtcdCountIfExternal(0),
				api.RemoveAllWorkerNodeGroups(),
			),
		),
	)
	runTinkerbellWorkloadClusterFlow(test)
}

func TestTinkerbellKubernetes122BottlerocketWorkloadClusterSkipPowerActions(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithBottleRocketTinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithNoPowerActions(),
			framework.WithControlPlaneHardware(2),
			framework.WithWorkerHardware(2),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithNoPowerActions(),
		),
	)
	runTinkerbellWorkloadClusterFlowSkipPowerActions(test)
}

func TestTinkerbellUpgradeMulticlusterWorkloadClusterWorkerScaleup(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithBottleRocketTinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithControlPlaneHardware(2),
			framework.WithWorkerHardware(3),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
		),
	)
	runSimpleWorkloadUpgradeFlowForBareMetal(
		test,
		v1alpha1.Kube122,
		framework.WithClusterUpgrade(
			api.WithKubernetesVersion(v1alpha1.Kube122),
			api.WithWorkerNodeCount(2),
		),
	)
}

func TestTinkerbellUpgradeMulticlusterWorkloadClusterCPScaleup(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithUbuntu122Tinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithControlPlaneHardware(4),
			framework.WithWorkerHardware(1),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
		),
	)
	runSimpleWorkloadUpgradeFlowForBareMetal(
		test,
		v1alpha1.Kube122,
		framework.WithClusterUpgrade(
			api.WithKubernetesVersion(v1alpha1.Kube122),
			api.WithControlPlaneCount(3),
		),
	)
}

func TestTinkerbellUpgradeMulticlusterWorkloadClusterWorkerScaleDown(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithBottleRocketTinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithControlPlaneHardware(1),
			framework.WithWorkerHardware(2),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithClusterFiller(api.WithWorkerNodeCount(2)),
		),
	)
	runSimpleWorkloadUpgradeFlowForBareMetal(
		test,
		v1alpha1.Kube122,
		framework.WithClusterUpgrade(
			api.WithKubernetesVersion(v1alpha1.Kube122),
			api.WithWorkerNodeCount(1),
		),
	)
}

func TestTinkerbellUpgradeMulticlusterWorkloadClusterK8sUpgrade(t *testing.T) {
	provider := framework.NewTinkerbell(t, framework.WithUbuntu122Tinkerbell())
	test := framework.NewMulticlusterE2ETest(
		t,
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
			framework.WithControlPlaneHardware(3),
			framework.WithWorkerHardware(3),
		),
		framework.NewClusterE2ETest(
			t,
			provider,
			framework.WithClusterFiller(api.WithKubernetesVersion(v1alpha1.Kube122)),
		),
	)
	runSimpleWorkloadUpgradeFlowForBareMetal(
		test,
		v1alpha1.Kube123,
		framework.WithClusterUpgrade(api.WithKubernetesVersion(v1alpha1.Kube123)),
		provider.WithProviderUpgrade(framework.UpdateTinkerbellUbuntuTemplate123Var()),
	)
}
