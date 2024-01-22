package kubepkg

import (
	kubepkg "github.com/octohelm/kubepkg/cuepkg/kubepkg"
)

#Storage: kubepkg.#Volume & {
	mountPath: "/etc/registry"

	type: "PersistentVolumeClaim"
	opt: claimName: "container-registry-storage"
	spec: {
		accessModes: ["ReadWriteOnce"]
		resources: requests: storage: "10Gi"
		storageClassName: "local-path"
	}
}
