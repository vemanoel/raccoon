## Development

1. Create your local environment configuration:
    ```bash
    cp .container/env.sample .container/env
    ```

2. Set RACCOON_BROWSER_PROFILE in .container/env to the absolute path of your browser profile directory:
    ```bash
    RACCOON_BROWSER_PROFILE=/path/to/browser/profile
    ```

2. Start the development environment:
    ```bash
    make up
    ```
