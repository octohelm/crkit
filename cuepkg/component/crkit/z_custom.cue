package crkit

#ContainerRegistry: spec: {
	volumes: "~container-registry-storage": #Storage
	services: "#": {
		clusterIP: *"10.68.0.255" | string
	}
	deploy: spec: template: spec: #DefaultNodeSelector
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
