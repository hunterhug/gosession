/*
	All right reserved：https://github.com/hunterhug/gosession at 2020
	Attribution-NonCommercial-NoDerivatives 4.0 International
	You can use it for education only but can't make profits for any companies and individuals!
*/
package gosession

import (
	"github.com/gofrs/uuid"
	"strconv"
	"strings"
)

// GetGUID gen random uuid
func GetGUID() (valueGUID string) {
	objID, _ := uuid.NewV4()
	objIdStr := objID.String()
	objIdStr = strings.Replace(objIdStr, "-", "", -1)
	valueGUID = objIdStr
	return valueGUID
}

func SI(s string) (i int64) {
	i, _ = strconv.ParseInt(s, 10, 64)
	return
}
