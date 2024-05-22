package crkit

import (
	kubepkg "github.com/octohelm/kubepkgspec/cuepkg/kubepkg"
)

#Storage: kubepkg.#Volume & {
	mountPath: "/etc/container-registry"

	type: "PersistentVolumeClaim"
	opt: claimName: "container-registry-storage"
	spec: {
		accessModes: ["ReadWriteOnce"]
		resources: requests: storage: "10Gi"
		storageClassName: "local-path"
	}
}

#ContainerRegistryServiceAccount: kubepkg.#ServiceAccount & {
	scope: "Namespace"
	rules: [
		{
			apiGroups: [""]
			resources: ["configmaps"]
			verbs: ["*"]
		},
		{
			apiGroups: [""]
			resources: ["nodes"]
			verbs: ["get"]
		},
	]
}

#ContainerdOperatorServiceAccount: kubepkg.#ServiceAccount & {
	scope: "Namespace"
	rules: [
		{
			apiGroups: [""]
			resources: ["configmaps"]
			verbs: ["*"]
		},
	]
}
