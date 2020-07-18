package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestProgramSpec defines the desired state of TestProgram
type TestProgramSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	InitialState  string   `json:"initialContext,omitempty"`
	StateStrategy Strategy `json:"stateStrategy"`
}

// TestProgramStatus defines the observed state of TestProgram
type TestProgramStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	CurrentState string `json:"context,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestProgram is the Schema for the testprograms API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=testprograms,scope=Namespaced
type TestProgram struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestProgramSpec   `json:"spec,omitempty"`
	Status TestProgramStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestProgramList contains a list of TestProgram
type TestProgramList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestProgram `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestProgram{}, &TestProgramList{})
}
