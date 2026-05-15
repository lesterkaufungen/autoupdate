package autoupdate

import _ "embed"

//go:embed scripts/update.sh
var updateScriptSh string

//go:embed scripts/update.ps1
var updateScriptPs1 string
