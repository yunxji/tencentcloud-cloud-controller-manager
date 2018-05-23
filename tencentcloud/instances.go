package tencentcloud

import (
	"context"
	"strings"
	"fmt"

	"github.com/dbdd4us/qcloudapi-sdk-go/cvm"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/kubernetes/pkg/cloudprovider"
)

// NodeAddresses returns the addresses of the specified instance.
// TODO(roberthbailey): This currently is only used in such a way that it
// returns the address of the calling instance. We should do a rename to
// make this clearer.
func (cloud *Cloud) NodeAddresses(ctx context.Context, name types.NodeName) ([]v1.NodeAddress, error) {
	addresses := make([]v1.NodeAddress, 0)

	privateIp, err := cloud.metadata.PrivateIPv4()
	if err == nil && privateIp == string(name) {
		addresses = append(addresses, v1.NodeAddress{Type: v1.NodeInternalIP, Address: privateIp})

		publicIp, err := cloud.metadata.PublicIPv4()
		if err != nil && len(publicIp) > 0 {
			addresses = append(addresses, v1.NodeAddress{Type: v1.NodeExternalIP, Address: publicIp})
		}
		return addresses, nil
	}

	// TODO query by node ip

	return []v1.NodeAddress{}, nil
}

// NodeAddressesByProviderID returns the addresses of the specified instance.
// The instance is specified using the providerID of the node. The
// ProviderID is a unique identifier of the node. This will not be called
// from the node whose nodeaddresses are being queried. i.e. local metadata
// services cannot be used in this method to obtain nodeaddresses
func (cloud *Cloud) NodeAddressesByProviderID(ctx context.Context, providerID string) ([]v1.NodeAddress, error) {
	id := strings.TrimPrefix(providerID, fmt.Sprintf("%s://", providerName))
	parts := strings.Split(id, "/")
	if len(parts) == 3 {
		instance, err := cloud.getInstanceByInstanceID(parts[2])
		if err != nil {
			return []v1.NodeAddress{}, err
		}
		addresses := make([]v1.NodeAddress, len(instance.PrivateIPAddresses)+len(instance.PublicIPAddresses))
		for idx, ip := range instance.PrivateIPAddresses {
			addresses[idx] = v1.NodeAddress{Type: v1.NodeInternalIP, Address: ip}
		}
		for idx, ip := range instance.PublicIPAddresses {
			addresses[len(instance.PrivateIPAddresses)+idx] = v1.NodeAddress{Type: v1.NodeExternalIP, Address: ip}
		}
		return addresses, nil
	}
	return []v1.NodeAddress{}, nil
}

// ExternalID returns the cloud provider ID of the node with the specified NodeName.
// Note that if the instance does not exist or is no longer running, we must return ("", cloudprovider.InstanceNotFound)
func (cloud *Cloud) ExternalID(ctx context.Context, nodeName types.NodeName) (string, error) {
	privateIp, err := cloud.metadata.PrivateIPv4()
	if err == nil && privateIp == string(nodeName) {
		instanceId, err := cloud.metadata.InstanceID()
		if err != nil {
			return "", err
		}
		// TODO add tencentcloud:// prefix?
		return instanceId, nil
	}

	// TODO query by node ip

	return "", nil
}

// InstanceID returns the cloud provider ID of the node with the specified NodeName.
func (cloud *Cloud) InstanceID(ctx context.Context, nodeName types.NodeName) (string, error) {
	privateIp, err := cloud.metadata.PrivateIPv4()
	if err == nil && privateIp == string(nodeName) {
		instanceId, err := cloud.metadata.InstanceID()
		if err != nil {
			return "", err
		}

		// TODO use metadata api or config
		zone, err := cloud.metadata.Zone()
		if err != nil {
			return "", err
		}
		return fmt.Sprintf("/%s/%s", zone, instanceId), nil
	}

	// TODO query node by ip
	return "", nil
}

// InstanceType returns the type of the specified instance.
func (cloud *Cloud) InstanceType(ctx context.Context, name types.NodeName) (string, error) {
	// TODO use tencentcloud?
	return providerName, nil
}

// InstanceTypeByProviderID returns the type of the specified instance.
func (cloud *Cloud) InstanceTypeByProviderID(ctx context.Context, providerID string) (string, error) {
	return providerName, nil
}

// AddSSHKeyToAllInstances adds an SSH public key as a legal identity for all instances
// expected format for the key is standard ssh-keygen format: <protocol> <blob>
func (cloud *Cloud) AddSSHKeyToAllInstances(ctx context.Context, user string, keyData []byte) error {
	return cloudprovider.NotImplemented
}

// CurrentNodeName returns the name of the node we are currently running on
// On most clouds (e.g. GCE) this is the hostname, so we provide the hostname
func (cloud *Cloud) CurrentNodeName(ctx context.Context, hostname string) (types.NodeName, error) {
	privateIp, err := cloud.metadata.PrivateIPv4()
	if err != nil {
		return types.NodeName(""), err
	}
	return types.NodeName(privateIp), nil
}

// InstanceExistsByProviderID returns true if the instance for the given provider id still is running.
// If false is returned with no error, the instance will be immediately deleted by the cloud controller manager.
func (cloud *Cloud) InstanceExistsByProviderID(ctx context.Context, providerID string) (bool, error) {
	return true, nil
}

func (cloud *Cloud) getInstanceByInstancePrivateIp(privateIp string) (*cvm.InstanceInfo, error) {
	instances, err := cloud.cvm.DescribeInstances(&cvm.DescribeInstancesArgs{
		Filters: &[]cvm.Filter{cvm.NewFilter(cvm.FilterNamePrivateIpAddress, privateIp)},
	})
	if err != nil {
		return nil, err
	}
	for _, instance := range instances.InstanceSet {
		for _, ip := range instance.PrivateIPAddresses {
			if ip == privateIp {
				return &instance, nil
			}
		}
	}

	return nil, cloudprovider.InstanceNotFound
}

func (cloud *Cloud) getInstanceByInstanceID(instanceID string) (*cvm.InstanceInfo, error) {
	instances, err := cloud.cvm.DescribeInstances(&cvm.DescribeInstancesArgs{
		Filters: &[]cvm.Filter{cvm.NewFilter(cvm.FilterNameInstanceId, instanceID)},
	})
	if err != nil {
		return nil, err
	}
	for _, instance := range instances.InstanceSet {
		if instance.InstanceID == instanceID {
			return &instance, nil
		}
	}

	return nil, cloudprovider.InstanceNotFound
}
