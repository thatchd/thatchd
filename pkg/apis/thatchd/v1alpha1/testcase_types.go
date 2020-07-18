package v1alpha1

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:validation:Enum=Created;Canceled;Dispatched;Running;Finished;Failed
type TestCaseCurrentStatus string

var (
	TestCaseCreated    TestCaseCurrentStatus = "Created"
	TestCaseCanceled   TestCaseCurrentStatus = "Canceled"
	TestCaseDispatched TestCaseCurrentStatus = "Dispatched"
	TestCaseRunning    TestCaseCurrentStatus = "Running"
	TestCaseFinished   TestCaseCurrentStatus = "Finished"
	TestCaseFailed     TestCaseCurrentStatus = "Failed"
)

const (
	DateTimeFormat = time.RFC822
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestCaseSpec defines the desired state of TestCase
type TestCaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Timeout  *string  `json:"timeout,omitempty"`
	Strategy Strategy `json:"strategy"`
}

// TestCaseStatus defines the observed state of TestCase
type TestCaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	DispatchedAt   *string               `json:"dispatchedAt,omitempty"`
	StartedAt      *string               `json:"startedAt,omitempty"`
	FinishedAt     *string               `json:"finishedAt,omitempty"`
	FailureMessage *string               `json:"failureMessage,omitempty"`
	Status         TestCaseCurrentStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestCase is the Schema for the testcases API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=testcases,scope=Namespaced
type TestCase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestCaseSpec   `json:"spec,omitempty"`
	Status TestCaseStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestCaseList contains a list of TestCase
type TestCaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestCase `json:"items"`
}

func TimeString(t time.Time) *string {
	result := t.Format(DateTimeFormat)
	return &result
}

func init() {
	SchemeBuilder.Register(&TestCase{}, &TestCaseList{})
}
