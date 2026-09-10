package v2

import (
	"github.com/ReCasaOS/CasaOS/codegen"
	"github.com/ReCasaOS/CasaOS/service"
)

type CasaOS struct {
	fileUploadService *service.FileUploadService
}

func NewCasaOS() codegen.ServerInterface {
	return &CasaOS{
		fileUploadService: service.NewFileUploadService(),
	}
}
