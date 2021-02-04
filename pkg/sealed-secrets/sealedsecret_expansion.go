package sealed_secrets

import (
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"fmt"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeserializer "k8s.io/apimachinery/pkg/runtime/serializer"

	"github.com/bitnami-labs/sealed-secrets/pkg/crypto"
)

const (
	// The StrictScope pins the sealed secret to a specific namespace and a specific name.
	StrictScope sealingScope = iota
	// The NamespaceWideScope only pins a sealed secret to a specific namespace.
	NamespaceWideScope
	// The ClusterWideScope allows the sealed secret to be unsealed in any namespace of the cluster.
	ClusterWideScope

	// The DefaultScope is currently the StrictScope.
	DefaultScope = StrictScope
)

// sealingScope is an enum that declares the mobility of a sealed secret by defining
// in which scopes
type sealingScope int

// encryptionLabel returns the label meant to be used for encrypting a sealed secret according to scope.
func encryptionLabel(namespace, name string, scope sealingScope) []byte {
	var l string
	switch scope {
	case ClusterWideScope:
		l = ""
	case NamespaceWideScope:
		l = namespace
	case StrictScope:
		fallthrough
	default:
		l = fmt.Sprintf("%s/%s", namespace, name)
	}
	return []byte(l)
}

// Returns labels followed by clusterWide followed by namespaceWide.
func labelFor(o metav1.Object) []byte {
	return encryptionLabel(o.GetNamespace(), o.GetName(), secretScope(o))
}

// secretScope returns the scope of a secret to be sealed, as annotated in its metadata.
func secretScope(o metav1.Object) sealingScope {
	if o.GetAnnotations()[SealedSecretClusterWideAnnotation] == "true" {
		return ClusterWideScope
	}
	if o.GetAnnotations()[SealedSecretNamespaceWideAnnotation] == "true" {
		return NamespaceWideScope
	}
	return StrictScope
}

// updateScopeAnnotations updates the annotation map so that it reflects the desired scope.
// It does so by updating and/or deleting existing annotations.
func updateScopeAnnotations(anno map[string]string, scope sealingScope) map[string]string {
	if anno == nil {
		anno = map[string]string{}
	}
	delete(anno, SealedSecretNamespaceWideAnnotation)
	delete(anno, SealedSecretClusterWideAnnotation)

	if scope == NamespaceWideScope {
		anno[SealedSecretNamespaceWideAnnotation] = "true"
	}
	if scope == ClusterWideScope {
		anno[SealedSecretClusterWideAnnotation] = "true"
	}
	return anno
}

// stripLastAppliedAnnotations strips annotations added by tools such as kubectl and kubecfg
// that contain a full copy of the original object kept in the annotation for strategic-merge-patch
// purposes. We need to remove these annotations when sealing an existing secret otherwise we'd leak
// the secrets.
func stripLastAppliedAnnotations(annotations map[string]string) {
	if annotations == nil {
		return
	}
	keys := []string{
		"kubectl.kubernetes.io/last-applied-configuration",
		"kubecfg.ksonnet.io/last-applied-configuration",
	}
	for _, k := range keys {
		delete(annotations, k)
	}
}

func parseSecret(secret *v1.Secret) *SealedSecret {
	return  &SealedSecret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "SealedSecret",
			APIVersion: "bitnami.com/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secret.GetName(),
			Namespace: secret.GetNamespace(),
		},
		Spec: SealedSecretSpec{
			Template: SecretTemplateSpec{
				// ObjectMeta copied below
				Type: secret.Type,
			},
			EncryptedData: map[string]string{},
		},
	}
}

// newSealedSecret creates a new SealedSecret object wrapping the
// provided secret. This encrypts only the values of each secrets
// individually, so secrets can be updated one by one.
func newSealedSecret(codecs runtimeserializer.CodecFactory, pubKey *rsa.PublicKey, secret *v1.Secret) (*SealedSecret, error) {
	if secretScope(secret) != ClusterWideScope && secret.GetNamespace() == "" {
		return nil, fmt.Errorf("secret must declare a namespace")
	}
	s:= parseSecret(secret)
	secret.ObjectMeta.DeepCopyInto(&s.Spec.Template.ObjectMeta)

	// the input secret could come from a real secret object applied with `kubectl apply` or similar tools
	// which put a copy of the object version at application time in an annotation in order to support
	// strategic merge patch in subsequent updates. We need to strip those annotations or else we would
	// be leaking secrets in clear in a way that might be non obvious to users.
	// See https://github.com/bitnami-labs/sealed-secrets/issues/227
	stripLastAppliedAnnotations(s.Spec.Template.ObjectMeta.Annotations)

	// Cleanup ownerReference (See #243)
	s.Spec.Template.ObjectMeta.OwnerReferences = nil

	// RSA-OAEP will fail to decrypt unless the same label is used
	// during decryption.
	label := labelFor(secret)

	for key, value := range secret.Data {
		ciphertext, err := crypto.HybridEncrypt(rand.Reader, pubKey, value, label)
		if err != nil {
			return nil, err
		}
		s.Spec.EncryptedData[key] = base64.StdEncoding.EncodeToString(ciphertext)
	}

	for key, value := range secret.StringData {
		ciphertext, err := crypto.HybridEncrypt(rand.Reader, pubKey, []byte(value), label)
		if err != nil {
			return nil, err
		}
		s.Spec.EncryptedData[key] = base64.StdEncoding.EncodeToString(ciphertext)
	}

	s.Annotations = updateScopeAnnotations(s.Annotations, secretScope(secret))

	return s, nil
}
