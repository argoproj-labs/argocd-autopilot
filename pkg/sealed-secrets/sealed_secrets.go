package sealed_secrets

import (
	"context"
	"crypto/rsa"
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/bitnami-labs/sealed-secrets/pkg/apis/sealed-secrets/v1alpha1"
	"github.com/codefresh-io/cf-argo/pkg/store"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/scheme"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/util/cert"
)

func CreateSealedSecretFromSecretFile(ctx context.Context, namespace, secretPath string, dryRun bool) (*v1alpha1.SealedSecret, error) {
	s, err := getSecretFromFile(ctx, secretPath)
	if err != nil {
		return nil, err
	}

	if dryRun {
		s.Data = nil
		s.StringData = nil
		ss, err := v1alpha1.NewSealedSecret(scheme.Codecs, nil, s)
		if err != nil {
			return nil, err
		}

		return addTypeMeta(ss), nil
	}

	rsaPub, err := getPubKey(ctx, namespace)
	if err != nil {
		return nil, err
	}

	ss, err := v1alpha1.NewSealedSecret(scheme.Codecs, rsaPub, s)
	if err != nil {
		return nil, err
	}

	return addTypeMeta(ss), nil
}

func addTypeMeta(ss *v1alpha1.SealedSecret) *v1alpha1.SealedSecret {
	ss.TypeMeta = metav1.TypeMeta{
		Kind:       "SealedSecret",
		APIVersion: v1alpha1.SchemeGroupVersion.String(),
	}
	return ss
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
