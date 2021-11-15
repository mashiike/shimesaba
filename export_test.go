package shimesaba

import gc "github.com/kayac/go-config"

func NewLoader(pathBase string) *gc.Loader {
	return newLoader(pathBase)
}
