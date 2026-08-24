package authority

import "github.com/nauticana/charter/sdk/model"

func effectiveNamespace(owner, explicit string) string {
	if explicit != "" {
		return explicit
	}
	return owner
}

func sameRef(leftOwner string, left model.Ref, rightOwner string, right model.Ref) bool {
	return left.ID == right.ID && effectiveNamespace(leftOwner, left.Namespace) == effectiveNamespace(rightOwner, right.Namespace)
}

func sameObjectRef(leftOwner string, left model.ObjectRef, rightOwner string, right model.ObjectRef) bool {
	return left.Kind == right.Kind && sameRef(leftOwner, model.Ref{Namespace: left.Namespace, ID: left.ID}, rightOwner, model.Ref{Namespace: right.Namespace, ID: right.ID})
}
