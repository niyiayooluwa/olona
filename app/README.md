# Olona

Flutter client for Olona, a personal inventory & ledger platform.

The app talks to the `server/` Go backend in this monorepo. See the repository
root for the backend, migrations, and API docs.

## Getting started

1. Ensure the backend is running (default `http://localhost:8080`). See
   `../server/README` and `../server/.env.example`.
2. Fetch dependencies and run:

   ```sh
   flutter pub get
   flutter run
   ```

## Useful commands

```sh
flutter analyze   # static analysis
flutter test      # run tests
```
