package investigation

import (
	"fmt"
)

type ClusterNotFoundError struct {
	ClusterID string
	Err       error
}

func (e ClusterNotFoundError) Error() string {
	return fmt.Sprintf("could not retrieve cluster info for %s: %s", e.ClusterID, e.Err.Error())
}

type ClusterDeploymentNotFoundError struct {
	ClusterID string
	Err       error
}

func (e ClusterDeploymentNotFoundError) Error() string {
	return fmt.Sprintf("could not retrieve clusterdeployment for %s: %s", e.ClusterID, e.Err.Error())
}

type AWSClientError struct {
	ClusterID string
	Err       error
}

func (e AWSClientError) Unwrap() error { return e.Err }

func (e AWSClientError) Error() string {
	return fmt.Sprintf("could not retrieve aws credentials for %s: %s", e.ClusterID, e.Err.Error())
}

type RestConfigError struct {
	ClusterID string
	Err       error
}

func (e RestConfigError) Unwrap() error { return e.Err }

func (e RestConfigError) Error() string {
	return fmt.Sprintf("could not create rest config for %s: %s", e.ClusterID, e.Err.Error())
}

type OCClientError struct {
	ClusterID string
	Err       error
}

func (e OCClientError) Unwrap() error { return e.Err }

func (e OCClientError) Error() string {
	return fmt.Sprintf("could not create oc client for %s: %s", e.ClusterID, e.Err.Error())
}

type K8SClientError struct {
	ClusterID string
	Err       error
}

func (e K8SClientError) Unwrap() error { return e.Err }

func (e K8SClientError) Error() string {
	return fmt.Sprintf("could not build k8s client for %s: %s", e.ClusterID, e.Err.Error())
}

type ManagementClusterNotFoundError struct {
	ClusterID string
	Err       error
}

func (e ManagementClusterNotFoundError) Error() string {
	return fmt.Sprintf("could not retrieve management cluster for HCP cluster %s: %s", e.ClusterID, e.Err.Error())
}

type ManagementRestConfigError struct {
	ClusterID           string
	ManagementClusterID string
	Err                 error
}

func (e ManagementRestConfigError) Unwrap() error { return e.Err }

func (e ManagementRestConfigError) Error() string {
	return fmt.Sprintf("could not create rest config for management cluster %s (HCP cluster: %s): %s", e.ManagementClusterID, e.ClusterID, e.Err.Error())
}

type ManagementK8sClientError struct {
	ClusterID           string
	ManagementClusterID string
	Err                 error
}

func (e ManagementK8sClientError) Unwrap() error { return e.Err }

func (e ManagementK8sClientError) Error() string {
	return fmt.Sprintf("could not create k8s client for management cluster %s (HCP cluster: %s): %s", e.ManagementClusterID, e.ClusterID, e.Err.Error())
}

type ManagementOCClientError struct {
	ClusterID           string
	ManagementClusterID string
	Err                 error
}

func (e ManagementOCClientError) Unwrap() error { return e.Err }

func (e ManagementOCClientError) Error() string {
	return fmt.Sprintf("could not create oc client for management cluster %s (HCP cluster: %s): %s", e.ManagementClusterID, e.ClusterID, e.Err.Error())
}
