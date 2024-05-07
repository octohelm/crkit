package crkit

#ContainerRegistry: spec: {
	serviceAccount: #ContainerRegistryServiceAccount
	deploy: spec: template: spec: #DefaultNodeSelector
	volumes: "~container-registry-storage": #Storage
}

#ContainerdOperator: spec: {
	serviceAccount: #ContainerdOperatorServiceAccount
}

#DefaultNodeSelector: {
	nodeSelector: {
		"node-role.kubernetes.io/control-plane": "true"
	}
	tolerations: [
		{
			key:      "node-role.kubernetes.io/master"
			operator: "Exists"
			effect:   "NoSchedule"
		},
	]
}
