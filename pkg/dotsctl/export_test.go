package dotsctl

// ResolveProfile exposes resolveProfile to the external test package.
func (d *Dots) ResolveProfile(name string) ([]ResolvedPackage, error) {
	return d.resolveProfile(name, nil)
}
