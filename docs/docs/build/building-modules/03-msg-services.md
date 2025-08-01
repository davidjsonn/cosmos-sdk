---
sidebar_position: 1
---

# `Msg` Services

:::note Synopsis
A Protobuf `Msg` service processes [messages](./02-messages-and-queries.md#messages). Protobuf `Msg` services are specific to the module in which they are defined, and only process messages defined within the said module. They are called from `BaseApp` during [`DeliverTx`](../../learn/advanced/00-baseapp.md#delivertx).
:::

:::note Pre-requisite Readings

* [Module Manager](./01-module-manager.md)
* [Messages and Queries](./02-messages-and-queries.md)

:::

## Implementation of a module `Msg` service

Each module should define a Protobuf `Msg` service, which will be responsible for processing requests (implementing `sdk.Msg`) and returning responses.

As further described in [ADR 031](../../../architecture/adr-031-msg-service.md), this approach has the advantage of clearly specifying return types and generating server and client code.

Protobuf generates a `MsgServer` interface based on a definition of `Msg` service. It is the role of the module developer to implement this interface, by implementing the state transition logic that should happen upon receival of each `sdk.Msg`. As an example, here is the generated `MsgServer` interface for `x/bank`, which exposes two `sdk.Msg`s:

```go reference
https://github.com/cosmos/cosmos-sdk/blob/v0.50.0-alpha.0/x/bank/types/tx.pb.go#L550-L568
```

When possible, the existing module's [`Keeper`](./06-keeper.md) should implement `MsgServer`, otherwise a `msgServer` struct that embeds the `Keeper` can be created, typically in `./keeper/msg_server.go`:

```go reference
https://github.com/cosmos/cosmos-sdk/blob/v0.50.0-alpha.0/x/bank/keeper/msg_server.go#L17-L19
```

`msgServer` methods can retrieve the `context.Context` from the `context.Context` parameter method using the `sdk.UnwrapSDKContext`:

```go reference
https://github.com/cosmos/cosmos-sdk/blob/v0.50.0-alpha.0/x/bank/keeper/msg_server.go#L56
```

`sdk.Msg` processing usually follows these 3 steps:

### Validation

The message server must perform all validation required (both *stateful* and *stateless*) to make sure the `message` is valid.
The `signer` is charged for the gas cost of this validation.

For example, a `msgServer` method for a `transfer` message should check that the sending account has enough funds to actually perform the transfer. 

It is recommended to implement all validation checks in a separate function that passes state values as arguments. This implementation simplifies testing. As expected, expensive validation functions charge additional gas. Example:

```go
ValidateMsgA(msg MsgA, now Time, gm GasMeter) error {
	if now.Before(msg.Expire) {
		return sdkerrrors.ErrInvalidRequest.Wrap("msg expired")
	}
	gm.ConsumeGas(1000, "signature verification")
	return signatureVerification(msg.Prover, msg.Data)
}
```

:::warning
Previously, the `ValidateBasic` method was used to perform simple and stateless validation checks.
This way of validating is deprecated, this means the `msgServer` must perform all validation checks.
:::

### State Transition

After the validation is successful, the `msgServer` method uses the [`keeper`](./06-keeper.md) functions to access the state and perform a state transition.

### Events 

Before returning, `msgServer` methods generally emit one or more [events](../../learn/advanced/08-events.md) by using the `EventManager` held in the `ctx`. Use the new `EmitTypedEvent` function that uses protobuf-based event types:

```go
ctx.EventManager().EmitTypedEvent(
	&group.EventABC{Key1: Value1,  Key2, Value2})
```

or the older `EmitEvent` function: 

```go
ctx.EventManager().EmitEvent(
	sdk.NewEvent(
		eventType,  // e.g. sdk.EventTypeMessage for a message, types.CustomEventType for a custom event defined in the module
		sdk.NewAttribute(key1, value1),
		sdk.NewAttribute(key2, value2),
	),
)
```

These events are relayed back to the underlying consensus engine and can be used by service providers to implement services around the application. Click [here](../../learn/advanced/08-events.md) to learn more about events.

The invoked `msgServer` method returns a `proto.Message` response and an `error`. These return values are then wrapped into an `*sdk.Result` or an `error` using `sdk.WrapServiceResult(ctx context.Context, res proto.Message, err error)`:

```go reference
https://github.com/cosmos/cosmos-sdk/blob/v0.50.0-alpha.0/baseapp/msg_service_router.go#L160
```

This method takes care of marshaling the `res` parameter to protobuf and attaching any events on the `ctx.EventManager()` to the `sdk.Result`.

```protobuf reference
https://github.com/cosmos/cosmos-sdk/blob/v0.50.0-alpha.0/proto/cosmos/base/abci/v1beta1/abci.proto#L93-L113
```

This diagram shows a typical structure of a Protobuf `Msg` service, and how the message propagates through the module.

![Transaction flow](https://raw.githubusercontent.com/cosmos/cosmos-sdk/release/v0.46.x/docs/uml/svg/transaction_flow.svg)

## Telemetry

New [telemetry metrics](../../learn/advanced/09-telemetry.md) can be created from `msgServer` methods when handling messages.

This is an example from the `x/auth/vesting` module:

```go reference
https://github.com/cosmos/cosmos-sdk/blob/v0.50.0-alpha.0/x/auth/vesting/msg_server.go#L76-L88
```
