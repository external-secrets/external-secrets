//Copyright External Secrets Inc. All Rights Reserved

package find

import (
	"fmt"
	"regexp"

	esv1beta1 "github.com/external-secrets/external-secrets/apis/externalsecrets/v1beta1"
)

type Matcher struct {
	re *regexp.Regexp
}

func New(findName esv1beta1.FindName) (*Matcher, error) {
	cmp, err := regexp.Compile(findName.RegExp)
	if err != nil {
		return nil, fmt.Errorf("could not compile find.name.regexp [%s]: %w", findName.RegExp, err)
	}
	return &Matcher{
		re: cmp,
	}, nil
}

func (m *Matcher) MatchName(name string) bool {
	return m.re.MatchString(name)
}
