/*
	All right reserved：https://github.com/hunterhug/gosession at 2020
	Attribution-NonCommercial-NoDerivatives 4.0 International
	You can use it for education only but can't make profits for any companies and individuals!
*/
package gosession

import (
	uuid "github.com/satori/go.uuid"
	"strings"
)

// gen random uuid
func GetGUID() (valueGUID string) {
	objID := uuid.NewV4()
	objIdStr := objID.String()
	objIdStr = strings.Replace(objIdStr, "-", "", -1)
	valueGUID = objIdStr
	return valueGUID
}
