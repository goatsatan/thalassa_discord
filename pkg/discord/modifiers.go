package discord

func (s *shardInstance) addServerInstance(guildID string, instance *ServerInstance) {
	s.Lock()
	s.serverInstances[guildID] = instance
	s.Unlock()
}

func (s *shardInstance) getServerInstance(guildID string) *ServerInstance {
	s.RLock()
	defer s.RUnlock()
	return s.serverInstances[guildID]
}

func (s *shardInstance) removeServerInstance(guildID string) {
	s.Lock()
	delete(s.serverInstances, guildID)
	s.Unlock()
}
