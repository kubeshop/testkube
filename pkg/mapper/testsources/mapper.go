package testsources

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	testsourcev1 "github.com/kubeshop/testkube-operator/api/testsource/v1"
	"github.com/kubeshop/testkube/pkg/api/v1/testkube"
)

// MapCRDToAPI maps TestSource CRD to OpenAPI spec TestSource
func MapCRDToAPI(item testsourcev1.TestSource) testkube.TestSource {
	var repository *testkube.Repository
	if item.Spec.Repository != nil {
		repository = &testkube.Repository{
			Type_:             item.Spec.Repository.Type_,
			Uri:               item.Spec.Repository.Uri,
			Branch:            item.Spec.Repository.Branch,
			Commit:            item.Spec.Repository.Commit,
			Path:              item.Spec.Repository.Path,
			WorkingDir:        item.Spec.Repository.WorkingDir,
			CertificateSecret: item.Spec.Repository.CertificateSecret,
			AuthType:          string(item.Spec.Repository.AuthType),
		}

		if item.Spec.Repository.UsernameSecret != nil {
			repository.UsernameSecret = &testkube.SecretRef{
				Name: item.Spec.Repository.UsernameSecret.Name,
				Key:  item.Spec.Repository.UsernameSecret.Key,
			}
		}

		if item.Spec.Repository.TokenSecret != nil {
			repository.TokenSecret = &testkube.SecretRef{
				Name: item.Spec.Repository.TokenSecret.Name,
				Key:  item.Spec.Repository.TokenSecret.Key,
			}
		}
	}

	return testkube.TestSource{
		Name:       item.Name,
		Namespace:  item.Namespace,
		Type_:      string(item.Spec.Type_),
		Uri:        item.Spec.Uri,
		Data:       item.Spec.Data,
		Repository: repository,
		Labels:     item.Labels,
	}
}

// MapAPIToCRD maps OpenAPI spec TestSourceUpsertRequest to CRD TestSource
func MapAPIToCRD(request testkube.TestSourceUpsertRequest) testsourcev1.TestSource {
	var repository *testsourcev1.Repository
	if request.Repository != nil {
		repository = &testsourcev1.Repository{
			Type_:             request.Repository.Type_,
			Uri:               request.Repository.Uri,
			Branch:            request.Repository.Branch,
			Commit:            request.Repository.Commit,
			Path:              request.Repository.Path,
			WorkingDir:        request.Repository.WorkingDir,
			CertificateSecret: request.Repository.CertificateSecret,
			AuthType:          testsourcev1.GitAuthType(request.Repository.AuthType),
		}

		if request.Repository.UsernameSecret != nil {
			repository.UsernameSecret = &testsourcev1.SecretRef{
				Name: request.Repository.UsernameSecret.Name,
				Key:  request.Repository.UsernameSecret.Key,
			}
		}

		if request.Repository.TokenSecret != nil {
			repository.TokenSecret = &testsourcev1.SecretRef{
				Name: request.Repository.TokenSecret.Name,
				Key:  request.Repository.TokenSecret.Key,
			}
		}
	}

	return testsourcev1.TestSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      request.Name,
			Namespace: request.Namespace,
			Labels:    request.Labels,
		},
		Spec: testsourcev1.TestSourceSpec{
			Type_:      testsourcev1.TestSourceType(request.Type_),
			Uri:        request.Uri,
			Data:       request.Data,
			Repository: repository,
		},
	}
}

// MapUpdateToSpec maps TestSourceUpdateRequest to TestSource CRD spec
func MapUpdateToSpec(request testkube.TestSourceUpdateRequest, testSource *testsourcev1.TestSource) *testsourcev1.TestSource {
	var fields = []struct {
		source      *string
		destination *string
	}{
		{
			request.Name,
			&testSource.Name,
		},
		{
			request.Namespace,
			&testSource.Namespace,
		},
		{
			request.Data,
			&testSource.Spec.Data,
		},
		{
			request.Uri,
			&testSource.Spec.Uri,
		},
	}

	for _, field := range fields {
		if field.source != nil {
			*field.destination = *field.source
		}
	}

	if request.Type_ != nil {
		testSource.Spec.Type_ = testsourcev1.TestSourceType(*request.Type_)
	}

	if request.Labels != nil {
		testSource.Labels = *request.Labels
	}

	if request.Repository != nil {
		if *request.Repository == nil {
			testSource.Spec.Repository = nil
			return testSource
		}

		if (*request.Repository).IsEmpty() {
			testSource.Spec.Repository = nil
			return testSource
		}

		if testSource.Spec.Repository == nil {
			testSource.Spec.Repository = &testsourcev1.Repository{}
		}

		empty := true
		fake := ""
		var fields = []struct {
			source      *string
			destination *string
		}{
			{
				(*request.Repository).Type_,
				&testSource.Spec.Repository.Type_,
			},
			{
				(*request.Repository).Uri,
				&testSource.Spec.Repository.Uri,
			},
			{
				(*request.Repository).Branch,
				&testSource.Spec.Repository.Branch,
			},
			{
				(*request.Repository).Commit,
				&testSource.Spec.Repository.Commit,
			},
			{
				(*request.Repository).Path,
				&testSource.Spec.Repository.Path,
			},
			{
				(*request.Repository).WorkingDir,
				&testSource.Spec.Repository.WorkingDir,
			},
			{
				(*request.Repository).CertificateSecret,
				&testSource.Spec.Repository.CertificateSecret,
			},
			{
				(*request.Repository).Username,
				&fake,
			},
			{
				(*request.Repository).Token,
				&fake,
			},
		}

		for _, field := range fields {
			if field.source != nil {
				*field.destination = *field.source
				empty = false
			}
		}

		if (*request.Repository).AuthType != nil {
			testSource.Spec.Repository.AuthType = testsourcev1.GitAuthType(*(*request.Repository).AuthType)
			empty = false
		}

		if (*request.Repository).UsernameSecret != nil {
			if (*(*request.Repository).UsernameSecret).IsEmpty() {
				testSource.Spec.Repository.UsernameSecret = nil
			} else {
				testSource.Spec.Repository.UsernameSecret = &testsourcev1.SecretRef{
					Name: (*(*request.Repository).UsernameSecret).Name,
					Key:  (*(*request.Repository).UsernameSecret).Key,
				}
			}

			empty = false
		}

		if (*request.Repository).TokenSecret != nil {
			if (*(*request.Repository).TokenSecret).IsEmpty() {
				testSource.Spec.Repository.TokenSecret = nil
			} else {
				testSource.Spec.Repository.TokenSecret = &testsourcev1.SecretRef{
					Name: (*(*request.Repository).TokenSecret).Name,
					Key:  (*(*request.Repository).TokenSecret).Key,
				}
			}

			empty = false
		}

		if empty {
			testSource.Spec.Repository = nil
		}
	}

	return testSource
}

// MapSpecToUpdate maps TestSource CRD spec TestSourceUpdateRequest to
func MapSpecToUpdate(testSource *testsourcev1.TestSource) (request testkube.TestSourceUpdateRequest) {
	var fields = []struct {
		source      *string
		destination **string
	}{
		{
			&testSource.Name,
			&request.Name,
		},
		{
			&testSource.Namespace,
			&request.Namespace,
		},
		{
			&testSource.Spec.Data,
			&request.Data,
		},
		{
			&testSource.Spec.Uri,
			&request.Uri,
		},
	}

	for _, field := range fields {
		*field.destination = field.source
	}

	request.Type_ = (*string)(&testSource.Spec.Type_)

	request.Labels = &testSource.Labels

	if testSource.Spec.Repository != nil {
		repository := &testkube.RepositoryUpdate{}
		request.Repository = &repository

		var fields = []struct {
			source      *string
			destination **string
		}{
			{
				&testSource.Spec.Repository.Type_,
				&(*request.Repository).Type_,
			},
			{
				&testSource.Spec.Repository.Uri,
				&(*request.Repository).Uri,
			},
			{
				&testSource.Spec.Repository.Branch,
				&(*request.Repository).Branch,
			},
			{
				&testSource.Spec.Repository.Commit,
				&(*request.Repository).Commit,
			},
			{
				&testSource.Spec.Repository.Path,
				&(*request.Repository).Path,
			},
			{
				&testSource.Spec.Repository.WorkingDir,
				&(*request.Repository).WorkingDir,
			},
			{
				&testSource.Spec.Repository.CertificateSecret,
				&(*request.Repository).CertificateSecret,
			},
		}

		for _, field := range fields {
			*field.destination = field.source
		}

		(*request.Repository).AuthType = (*string)(&testSource.Spec.Repository.AuthType)

		if testSource.Spec.Repository.UsernameSecret != nil {
			secretRef := &testkube.SecretRef{
				Name: testSource.Spec.Repository.UsernameSecret.Name,
				Key:  testSource.Spec.Repository.UsernameSecret.Key,
			}

			(*request.Repository).UsernameSecret = &secretRef
		}

		if testSource.Spec.Repository.TokenSecret != nil {
			secretRef := &testkube.SecretRef{
				Name: testSource.Spec.Repository.TokenSecret.Name,
				Key:  testSource.Spec.Repository.TokenSecret.Key,
			}

			(*request.Repository).TokenSecret = &secretRef
		}
	}

	return request
}
