package sealed_secrets

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/codefresh-io/cf-argo/pkg/store"
	apiv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/cert"
	"k8s.io/kubectl/pkg/scheme"
)

// GroupName is the group name used in this package

const (
	GroupName = "bitnami.com"
	// SealedSecretName is the name used in SealedSecret CRD
	SealedSecretName = "sealed-secret." + GroupName
	// SealedSecretPlural is the collection plural used with SealedSecret API
	SealedSecretPlural = "sealedsecrets"

	// Annotation namespace prefix
	annoNs = "sealedsecrets." + GroupName + "/"

	// SealedSecretClusterWideAnnotation is the name for the annotation for
	// setting the secret to be available cluster wide.
	SealedSecretClusterWideAnnotation = annoNs + "cluster-wide"

	// SealedSecretNamespaceWideAnnotation is the name for the annotation for
	// setting the secret to be available namespace wide.
	SealedSecretNamespaceWideAnnotation = annoNs + "namespace-wide"

	// SealedSecretManagedAnnotation is the name for the annotation for
	// flaging the existing secrets be managed by SealedSecret controller.
	SealedSecretManagedAnnotation = annoNs + "managed"
)

// SecretTemplateSpec describes the structure a Secret should have
// when created from a template
type SecretTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"bytes,1,opt,name=metadata"`

	// Used to facilitate programmatic handling of secret data.
	// +optional
	Type apiv1.SecretType `json:"type,omitempty" protobuf:"bytes,3,opt,name=type,casttype=SecretType"`
}

// SealedSecretSpec is the specification of a SealedSecret
type SealedSecretSpec struct {
	// Template defines the structure of the Secret that will be
	// created from this sealed secret.
	// +optional
	Template SecretTemplateSpec `json:"template,omitempty"`

	// Data is deprecated and will be removed eventually. Use per-value EncryptedData instead.
	Data          []byte            `json:"data,omitempty"`
	EncryptedData map[string]string `json:"encryptedData"`
}

// SealedSecretConditionType describes the type of SealedSecret condition
type SealedSecretConditionType string

const (
	// SealedSecretSynced means the SealedSecret has been decrypted and the Secret has been updated successfully.
	SealedSecretSynced SealedSecretConditionType = "Synced"
)

// SealedSecretCondition describes the state of a sealed secret at a certain point.
type SealedSecretCondition struct {
	// Type of condition for a sealed secret.
	// Valid value: "Synced"
	Type SealedSecretConditionType `json:"type" protobuf:"bytes,1,opt,name=type,casttype=DeploymentConditionType"`
	// Status of the condition for a sealed secret.
	// Valid values for "Synced": "True", "False", or "Unknown".
	Status apiv1.ConditionStatus `json:"status" protobuf:"bytes,2,opt,name=status,casttype=k8s.io/api/core/v1.ConditionStatus"`
	// The last time this condition was updated.
	LastUpdateTime metav1.Time `json:"lastUpdateTime,omitempty" protobuf:"bytes,6,opt,name=lastUpdateTime"`
	// Last time the condition transitioned from one status to another.
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty" protobuf:"bytes,7,opt,name=lastTransitionTime"`
	// The reason for the condition's last transition.
	Reason string `json:"reason,omitempty" protobuf:"bytes,4,opt,name=reason"`
	// A human readable message indicating details about the transition.
	Message string `json:"message,omitempty" protobuf:"bytes,5,opt,name=message"`
}

// SealedSecretStatus is the most recently observed status of the SealedSecret.
type SealedSecretStatus struct {
	// ObservedGeneration reflects the generation most recently observed by the sealed-secrets controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty" protobuf:"varint,3,opt,name=observedGeneration"`

	// Represents the latest available observations of a sealed secret's current state.
	// +optional
	// +patchMergeKey=type
	// +patchStrategy=merge
	Conditions []SealedSecretCondition `json:"conditions,omitempty" patchStrategy:"merge" patchMergeKey:"type" protobuf:"bytes,6,rep,name=conditions"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +genclient

// SealedSecret is the K8s representation of a "sealed Secret" - a
// regular k8s Secret that has been sealed (encrypted) using the
// controller's key.
type SealedSecret struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec SealedSecretSpec `json:"spec"`
	// +optional
	Status *SealedSecretStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SealedSecretList represents a list of SealedSecrets
type SealedSecretList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	Items []SealedSecret `json:"items"`
}

// ByCreationTimestamp is used to sort a list of secrets
type ByCreationTimestamp []apiv1.Secret

func (s ByCreationTimestamp) Len() int {
	return len(s)
}

func (s ByCreationTimestamp) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

func (s ByCreationTimestamp) Less(i, j int) bool {
	return s[i].GetCreationTimestamp().Unix() < s[j].GetCreationTimestamp().Unix()
}

func CreateSealedSecretFromSecretFile(ctx context.Context, namespace, secretPath string) (*SealedSecret, error) {
	rsaPub, err := getPubKey(ctx, namespace)
	if err != nil {
		return nil, err
	}

	s, err := getSecretFromFile(ctx, secretPath)
	if err != nil {
		return nil, err
	}

	return newSealedSecret(scheme.Codecs, rsaPub, s)
}

func getSecretFromFile(ctx context.Context, secretPath string) (*v1.Secret, error) {
	d := scheme.Codecs.UniversalDeserializer()
	bytes, err := ioutil.ReadFile(secretPath)
	if err != nil {
		return nil, err
	}

	o, _, err := d.Decode(bytes, nil, nil)
	if err != nil {
		return nil, err
	}

	switch s := o.(type) {
	case *v1.Secret:
		return s, nil
	default:
		k := s.GetObjectKind().GroupVersionKind()
		return nil, fmt.Errorf("unexpected runtime object of type: %s", k)
	}
}

func getPubKey(ctx context.Context, ns string) (*rsa.PublicKey, error) {
	conf, err := store.Get().NewKubeClient(ctx).ToRESTConfig()
	if err != nil {
		return nil, err
	}

	restClient, err := corev1.NewForConfig(conf)
	if err != nil {
		return nil, err
	}

	f, err := restClient.
		Services(ns).
		ProxyGet("http", "sealed-secrets-controller", "", "/v1/cert.pem", nil).
		Stream(ctx)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseKey(f)
}

func parseKey(r io.Reader) (*rsa.PublicKey, error) {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}
	certs, err := cert.ParseCertsPEM(data)
	if err != nil {
		return nil, err
	}
	// ParseCertsPem returns error if len(certs) == 0, but best to be sure...
	if len(certs) == 0 {
		return nil, errors.New("Failed to read any certificates")
	}
	cert, ok := certs[0].PublicKey.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("Expected RSA public key but found %v", certs[0].PublicKey)
	}
	return cert, nil
}
