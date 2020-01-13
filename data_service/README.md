### Database

The `db/` directory contains all the code necessary
to Wiregost for interacting with the PostgreSQL database.
It all also contains all entities used in Wiregost that will 
need some form of persistence.
This package should *not* import any other package from Wiregost,
maybe except from some configuration.

* `models/`     - All entities that need persistence in Wiregost, and their DB methods.
* `remote/`     - Methods used by client to make requests to data service
* `handlers/`   - Server handlers, calling models and their methods.
* `.config`     - Configuration file for database access and http service
* `main.go`     - Entry point for data service
