# What is this?

This project demonstrates how to build a CRDT (conflict-free replicated data structure) using gossip protocol.

# How to run this

```
docker compose up --build --force-recreate --scale node=3
```

This will bring up 3 nodes, each will have the current value of the counter. If one of the nodes increments its,
everyone else will get the same value of the counter _eventually_.

You can test this by stopping one of the container and restarting it after some time, you will see that it will have the
most recent value of the counter.