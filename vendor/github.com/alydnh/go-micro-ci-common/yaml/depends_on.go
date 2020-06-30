package yaml

import (
	"fmt"
	"github.com/alydnh/go-micro-ci-common/utils"
	"strings"
)

type dependsOnContext struct {
	processed   []*Service
	unprocessed []*Service
	chain       []*Service
}

func (c *dependsOnContext) setProcessed(service *Service) {
	for index, unprocessed := range c.unprocessed {
		if unprocessed == service {
			c.unprocessed = append(c.unprocessed[:index], c.unprocessed[index+1:]...)
			c.processed = append(c.processed, unprocessed)
			break
		}
	}
}

func (c *dependsOnContext) add(service *Service) {
	c.unprocessed = append(c.unprocessed, service)
}

func (c *dependsOnContext) inChain(service *Service) bool {
	return utils.Any(c.chain, func(s *Service) bool { return s.name == service.name })
}

func (c *dependsOnContext) chainPush(service *Service) {
	c.chain = append(c.chain, service)
}

func (c *dependsOnContext) chainPop() *Service {
	service := c.chain[len(c.chain)-1]
	c.chain = c.chain[0 : len(c.chain)-1]
	return service
}

func (c *dependsOnContext) chainString() string {
	return strings.Join(utils.Select(c.chain, func(s *Service) string { return s.name }).([]string), "->")
}

func (c *dependsOnContext) finished() bool {
	return len(c.unprocessed) <= 0
}

func (y *Service) processDependsOn(context *dependsOnContext) error {
	if utils.EmptyArray(y.DependsOn) {
		context.setProcessed(y)
		return nil
	}

	if context.inChain(y) {
		return fmt.Errorf("%s被循环依赖:%s", y.name, context.chainString())
	}

	context.chainPush(y)
	defer context.chainPop()
	for _, depend := range y.DependsOn {
		if _, ok := utils.FirstOrDefault(context.processed, func(s *Service) bool { return s.name == depend }); !ok {
			if v, ok := utils.FirstOrDefault(context.unprocessed, func(s *Service) bool { return s.name == depend }); !ok {
				return fmt.Errorf("%s.dependsOn[%s]未找到", y.name, depend)
			} else {
				service := v.(*Service)
				if err := service.processDependsOn(context); nil != err {
					return err
				}
			}
		}
	}

	context.setProcessed(y)
	return nil
}
