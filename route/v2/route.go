package v2

import (
	"github.com/inkly/CasaOS/codegen"
	"github.com/inkly/CasaOS/service"
)

type CasaOS struct {
	fileUploadService *service.FileUploadService
}

func NewCasaOS() codegen.ServerInterface {
	return &CasaOS{
		fileUploadService: service.NewFileUploadService(),
	}
}
