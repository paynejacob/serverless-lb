package resolver

type AddressPool struct {
	cursor    int
	addresses []string
}

func (a *AddressPool) AddAddress(addr string) {
	for i := range a.addresses {
		if a.addresses[i] == addr {
			return
		}
	}

	a.addresses = append(a.addresses, addr)
}

func (a *AddressPool) RemoveAddress(addr string) {
	var i int
	var found bool
	for i = range a.addresses {
		if a.addresses[i] == addr {
			found = true
			break
		}
	}

	if !found || len(a.addresses) == 0 {
		return
	}

	a.addresses[i] = a.addresses[len(a.addresses)-1]
	a.addresses = a.addresses[:len(a.addresses)-1]
	a.cursor = 0
}

func (a *AddressPool) Next() string {
	if len(a.addresses) == 0 {
		return ""
	}

	a.cursor += 1
	if a.cursor > len(a.addresses)-1 {
		a.cursor = 0
	}

	return a.addresses[a.cursor]
}
