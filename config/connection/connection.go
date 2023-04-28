package connection

import (
	common2 "cicd-core/config/common"
)

func Value(connName string) string {
	v := common2.NewManager().Get("/connection/" + connName)

	if vv, ok := v.(string); ok {
		return vv
	}

	return ""
}
