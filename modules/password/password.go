// Copyright 2019 The Gitea Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package password

import (
	"crypto/rand"
	"math/big"
	"strings"
	"sync"

	"code.gitea.io/gitea/modules/setting"
)

// Complexity contains information about a particular kind of password complexity
type Complexity struct {
	ValidChars string
	TrNameOne  string
}

var (
	matchComplexityOnce sync.Once
	validChars          string
	requiredList        []Complexity

	charComplexities = map[string]Complexity{
		"lower": {
			`abcdefghijklmnopqrstuvwxyz`,
			"password_lowercase_one",
		},
		"upper": {
			`ABCDEFGHIJKLMNOPQRSTUVWXYZ`,
			"password_uppercase_one",
		},
		"digit": {
			`0123456789`,
			"password_digit_one",
		},
		"spec": {
			` !"#$%&'()*+,-./:;<=>?@[\]^_{|}~` + "`",
			"password_special_one",
		},
	}
)

// NewComplexity for preparation
func NewComplexity() {
	matchComplexityOnce.Do(func() {
		setupComplexity(setting.PasswordComplexity)
	})
}

func setupComplexity(values []string) {
	if len(values) != 1 || values[0] != "off" {
		for _, val := range values {
			if complex, ok := charComplexities[val]; ok {
				validChars += complex.ValidChars
				requiredList = append(requiredList, complex)
			}
		}
		if len(requiredList) == 0 {
			// No valid character classes found; use all classes as default
			for _, complex := range charComplexities {
				validChars += complex.ValidChars
				requiredList = append(requiredList, complex)
			}
		}
	}
	if validChars == "" {
		// No complexities to check; provide a sensible default for password generation
		validChars = charComplexities["lower"].ValidChars + charComplexities["upper"].ValidChars + charComplexities["digit"].ValidChars
	}
}

// IsComplexEnough return True if password meets complexity settings
func IsComplexEnough(pwd string) bool {
	NewComplexity()
	if len(validChars) > 0 {
		for _, req := range requiredList {
			if !strings.ContainsAny(req.ValidChars, pwd) {
				return false
			}
		}
	}
	return true
}

// Generate  a random password
func Generate(n int) (string, error) {
	NewComplexity()
	buffer := make([]byte, n)
	max := big.NewInt(int64(len(validChars)))
	for {
		for j := 0; j < n; j++ {
			rnd, err := rand.Int(rand.Reader, max)
			if err != nil {
				return "", err
			}
			buffer[j] = validChars[rnd.Int64()]
		}
		if IsComplexEnough(string(buffer)) && string(buffer[0]) != " " && string(buffer[n-1]) != " " {
			return string(buffer), nil
		}
	}
}

// GetActiveComplexities returns a list of the active complexities (may differ from the settings)
func GetActiveComplexities() []Complexity {
	return requiredList
}
