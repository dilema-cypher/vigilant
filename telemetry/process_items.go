package telemetry

import (
	"context"
	"errors"
)



func AddNewEvent(ctx context.Context,key string, value any)error{
	evt := FromContext(ctx)

	if evt != nil {
		evt.Add(key,value)
		return nil
	}

	return errors.New("event not found")

}

func AddNewError(ctx context.Context,err error)error{
	evt := FromContext(ctx)

	if evt != nil {
		evt.AddError(err)
		return nil
	}

	return errors.New("event not found")
}

func ProcessItems(
	ctx context.Context,
	method string,
	url string,
	) error {

		evt := FromContext(ctx)

		if evt != nil{
			defer evt.End()

			evt.Add("method", method)
			evt.Add("url", url)
			return nil
		}
		
		return errors.New("event not found")
}