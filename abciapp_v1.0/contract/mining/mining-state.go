package mining

//_miningStartHeight This is a get mining start height method of Mining
func (m *Mining) miningStartHeight_() int64 {
	v := m.getMiningStartHeight()

	return v
}

//_setMiningStartHeight This is a set mining start height method of Mining
func (m *Mining) setMiningStartHeight_(v int64) {
	m.setMiningStartHeight(v)

}

//_miningRewardAmount This is a get mining reward amount method of Mining
func (m *Mining) miningRewardAmount_() uint64 {
	v := m.getMiningRewardAmount()

	return v
}

//_setMiningRewardAmount This is a set mining reward amount method of Mining
func (m *Mining) setMiningRewardAmount_(v uint64) {
	m.setMiningRewardAmount(v)

}
