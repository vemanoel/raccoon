## Development

1. Create your local environment configuration:
    ```bash
    cp .container/env.example .container/env
    ```

2. Open `.container/env` and set the `RACCOON_BROWSER_PROFILE` variable to the absolute path of your local browser profile directory:
    ```bash
    RACCOON_BROWSER_PROFILE=/path/to/browser/profile
    ```

2. Build and start all required containerized services:
    ```bash
    make up
    ```
