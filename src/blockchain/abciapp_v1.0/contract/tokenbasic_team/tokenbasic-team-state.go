package tokenbasic_team

func (t *TBTeam) global_() *TBTGlobal {
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

func (t *TBTeam) setGlobal_(v *TBTGlobal) {
	t.global__ = v
	t.setGlobalDB(v)
}
