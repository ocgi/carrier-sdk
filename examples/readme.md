# A example webhook server

## Usage

- check ready

    ```shell
    curl -X POST 127.0.0.1:8000/ready -d '{"request": {"uid": "123", "name": "testx", "namespace": "default"}}'
    ```
    ```shell
    {
      "request": {
        "uid": "123",
        "name": "testx",
        "namespace": "default"
      },
      "response": {
        "uid": "123",
        "allowed": false,
        "reason": "not allowed"
      }
    }
    ```
  
- set status

    ```shell
    curl -X POST 127.0.0.1:8000/set -d '{"name": "testx", "namespace": "default"}'|jq .
    ```
    ```shell
    {
        "status": "ok"
    }
    ```  

  
- check ready again

    ```shell
    curl -X POST 127.0.0.1:8000/ready -d '{"request": {"uid": "123", "name": "testx", "namespace": "default"}}'
    ```
    ```shell
    {
      "request": {
        "uid": "123",
        "name": "testx",
        "namespace": "default"
      },
      "response": {
        "uid": "123",
        "allowed": true
      }
    }
    ```