package domain

// AuxConfigResource identifies the supplemental UE configuration domains.
type AuxConfigResource string

const (
	ModelAnchors         AuxConfigResource = "model-anchors"
	SensorDevices        AuxConfigResource = "sensor-devices"
	ClusterAgentsCatalog AuxConfigResource = "cluster-agents-catalog"
	ClusterAgentRegistry AuxConfigResource = "cluster-agent-registry"
)

func (r AuxConfigResource) IDField() string {
	switch r {
	case ModelAnchors:
		return "anchor_id"
	case SensorDevices:
		return "device_id"
	case ClusterAgentsCatalog, ClusterAgentRegistry:
		return "nodeId"
	default:
		return ""
	}
}

func (r AuxConfigResource) Collection() string {
	switch r {
	case ModelAnchors:
		return "anchors"
	case SensorDevices:
		return "devices"
	case ClusterAgentsCatalog, ClusterAgentRegistry:
		return "agents"
	default:
		return ""
	}
}
