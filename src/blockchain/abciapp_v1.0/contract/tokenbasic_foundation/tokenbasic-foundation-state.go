package tokenbasic_foundation

func (t *TBFoundation) global_() *TBFGlobal {
	if t.global__ != nil {
		return t.global__
	}

	var err error
	t.global__, err = t.getGlobalDB()
	if err != nil {
		panic(err)
	}
	return t.global__
}

func (t *TBFoundation) setGlobal_(v *TBFGlobal) {
	t.global__ = v
	t.setGlobalDB(v)
}
