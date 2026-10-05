package sprig

import sv2 "github.com/Masterminds/semver/v3"

func semverCompare(constraint, version string) (bool, error) {
	c, err := sv2.NewConstraint(constraint)
	if err != nil {
		return false, errInvalidConstraint
	}
	v, err := sv2.NewVersion(version)
	if err != nil {
		return false, errInvalidVersion
	}
	return c.Check(v), nil
}
func semver(version string) (*sv2.Version, error) {
	v, err := sv2.NewVersion(version)
	if err != nil {
		return nil, errInvalidVersion
	}
	return v, nil
}
