package challenge

import (
	"errors"
	"fmt"
	"math/rand"
	"sync"
)

type Instances struct {
	Client             *Client
	challengeInstances map[Spec][]Instance
	userInstances      map[string]Instance
	mutex              *sync.Mutex
	bindIp             string
}

type Instance struct {
	Port      string
	Challenge Spec
	Users     []string
	Container string
	Stopped   bool
	avoiding  []string
}

func contains(haystack []string, needle string) bool {
	for _, element := range haystack {
		if element == needle {
			return true
		}
	}
	return false
}

func (i *Instances) GetChallengeByName(name string) Spec {
	return challengeSpecs[name]
}

func (i *Instances) GetInstanceForUser(user string, challenge Spec) (Instance, error) {
	if instance, ok := i.userInstances[user]; ok && instance.Challenge == challenge {
		return instance, nil
	}

	instances, ok := i.challengeInstances[challenge]
	if !ok || len(instances) == 0 {
		return Instance{}, errors.New("no instances of challenge")
	}

	for _, instance := range instances {
		if !instance.Stopped && len(instance.Users) < instance.Challenge.UserLimit && len(instance.avoiding) < 50 && !contains(instance.avoiding, user) {
			instance.Users = append(instance.Users, user)
			i.userInstances[user] = instance
			return instance, nil
		}
	}

	shuffledInstances := make([]Instance, len(instances))
	copy(shuffledInstances, instances)
	rand.Shuffle(len(instances), func(i, j int) {
		shuffledInstances[i], shuffledInstances[j] = shuffledInstances[j], shuffledInstances[i]
	})

	//This isn't ideal, however if every instance is full, its really the only option
	return shuffledInstances[0], nil
}

func (i *Instances) GetCurrentUserInstance(user string) Instance {
	if instance, ok := i.userInstances[user]; ok {
		return instance
	}
	return Instance{}
}

func (i *Instances) StartInstance(challenge Spec) {
	fmt.Println("Starting an instance of", challenge.Name)
	instance, err := i.Client.StartContainer(challenge, i.bindIp)
	if err != nil {
		fmt.Println(err)
		return
	}

	i.mutex.Lock()
	instances, ok := i.challengeInstances[challenge]
	if !ok {
		instances = make([]Instance, 1)
	}
	i.challengeInstances[challenge] = append(instances, instance)
	i.mutex.Unlock()
}

func (i *Instances) AvoidInstance(user string, instance Instance) {
	fmt.Println(instance)
	instance.avoiding = append(instance.avoiding, user)
}

func (i *Instances) Disconnect(user string) {
	instance := i.GetCurrentUserInstance(user)
	if instance.Users == nil {
		return
	}

	index := -1
	for j, userElement := range instance.Users {
		if user == userElement {
			index = j
			break
		}
	}
	if index != -1 {
		instance.Users = append(instance.Users[:index], instance.Users[index+1:]...)
	}
}

func (i *Instances) StopInstance(instance Instance) {
	_ = i.Client.StopContainer(instance.Container)
	instance.Stopped = true
}
