# Security

Report vulnerabilities privately to the repository maintainer. Do not include
live credentials, private OMEMO identity keys, or decrypted messages in public
issues. The library refuses SASL PLAIN on an unencrypted stream unless explicitly
configured otherwise. The built-in OMEMO package supplies protocol integration;
a wire-compatible Signal/Double-Ratchet backend must be supplied by the
application for production interoperability.
