package utils

import "golang.org/x/crypto/bcrypt"

// HashPassword 密码加盐哈希加密(生成不同盐值，防御彩虹表攻击)
func HashPassword(password string) (string, error) {
	//DefaultCost 是10， 兼顾安全与性能
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPasswordHash 检验明文密码与哈希密闻是否匹配
func CheckPasswordHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
