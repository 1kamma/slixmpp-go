# Contributing

Run `make check` before submitting changes. New protocol behavior should include
round-trip XML tests and, where possible, an in-memory client/server test. Do not
advertise a service-discovery feature until the corresponding behavior is
implemented and tested. Security-sensitive changes to TLS, SASL, stream
management, or OMEMO require focused tests and documentation updates.
