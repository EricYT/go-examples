package lazy_calcuate

import "log"

type InfiniteList struct {
	Head int
	Tail func() InfiniteList
}

func Generate(seed int, step func(int) int) InfiniteList {
	return InfiniteList{
		Head: seed,
		Tail: func() InfiniteList {
			log.Printf("Generate seed: %d go\n", seed)
			return Generate(step(seed), step)
		},
	}
}

func GenHead(list InfiniteList) int { return list.Head }

func GenTail(list InfiniteList) InfiniteList {
	return list.Tail()
}
