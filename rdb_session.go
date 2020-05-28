/*
	All right reserved：https://github.com/hunterhug/gosession at 2020
	Attribution-NonCommercial-NoDerivatives 4.0 International
	You can use it for education only but can't make profits for any companies and individuals!
*/
package gosession

// need todo
// session by mysql
type RdbSession struct {
	getUserFunc  func(id string) (*User, error) // when not hit cache will get user from this func
	tokenKey     string                         // prefix of token，default 'got'
	userKey      string                         // prefix of user info cache ，default 'gou'
	expireTime   int64                          // token expire how much second，default  7 days
	isSingleMode bool                           // is single token, new token will destroy other token
}
