package discord

func (s *ShardInstance) addServerInstance(guildID string, instance *ServerInstance) {
	s.Lock()
	s.ServerInstances[guildID] = instance
	s.Unlock()
}

func (s *ShardInstance) getServerInstance(guildID string) *ServerInstance {
	s.RLock()
	defer s.RUnlock()
	return s.ServerInstances[guildID]
}

func (s *ShardInstance) removeServerInstance(guildID string) {
	s.Lock()
	delete(s.ServerInstances, guildID)
	s.Unlock()
}
