//Copyright External Secrets Inc. All Rights Reserved

package v1alpha1

type ControllerClassResource struct {
	Spec struct {
		ControllerClass string `json:"controller"`
	} `json:"spec"`
}
