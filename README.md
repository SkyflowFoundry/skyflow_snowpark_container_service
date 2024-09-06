# Skyflow Detokenize Snowpark Container Service

This is a simple Go web server that performs bulk detokenization requests with Skyflow. The intention
is to run this using Snowpark Container Services in Snowflake to perform bulk detokenization from a
query written in Snowflake.

## Skyflow Detokenize Snowpark Container Service Script

Before you can test the code, please follow the instructions for [Snowpark Container Services](https://docs.snowflake.com/en/developer-guide/snowpark-container-services/overview) to get the Docker container deployed.

The code below shows how to setup the container services.
``` sql
CREATE ROLE test_role;

-- Enable OAuth services for the Snowflake account. Needed in order to login before navigating to the service URL.
CREATE SECURITY INTEGRATION IF NOT EXISTS snowservices_ingress_oauth
  TYPE=oauth
  OAUTH_CLIENT=snowservices_ingress
  ENABLED=true;

-- Give this account access to create a service endpoint.
GRANT BIND SERVICE ENDPOINT ON ACCOUNT TO ROLE test_role;

-- Create the virtual machine we'll run the service on.
CREATE COMPUTE POOL detokenize_compute_pool
  MIN_NODES = 1
  MAX_NODES = 1
  INSTANCE_FAMILY = CPU_X64_XS; -- 2 vCPU, 8GB memory

CREATE OR REPLACE NETWORK RULE ALLOW_BACKEND_RULE
  TYPE = HOST_PORT
  MODE = EGRESS
  VALUE_LIST= ('manage.skyflowapis.com', '<REPLACE_ME_WITH_VAUL_URL>');

CREATE OR REPLACE EXTERNAL ACCESS INTEGRATION ALLOW_BACKEND_INTEGRATION
  ALLOWED_NETWORK_RULES = (ALLOW_BACKEND_RULE)
  ENABLED = true;

USE ROLE accountadmin;
USE DATABASE detokenize_db;
  
GRANT USAGE ON NETWORK RULE ALLOW_BACKEND_RULE TO ROLE test_role;
GRANT USAGE ON INTEGRATION ALLOW_BACKEND_INTEGRATION TO ROLE test_role;

-- Create a warehouse in case we want run SQL statements against the service.
CREATE OR REPLACE WAREHOUSE detokenize_warehouse WITH
  WAREHOUSE_SIZE='X-Small'
  AUTO_SUSPEND = 180
  AUTO_RESUME = true
  INITIALLY_SUSPENDED=false;

GRANT ALL ON WAREHOUSE detokenize_warehouse TO ROLE test_role;
GRANT ALL ON COMPUTE POOL detokenize_compute_pool TO ROLE test_role;

-- Make sure we're using the correct role, database, and warehouse.
USE ROLE test_role;
USE DATABASE detokenize_db;
USE WAREHOUSE detokenize_warehouse;

-- Create a schema.
CREATE SCHEMA IF NOT EXISTS data_schema;

-- Create the image repository within the image registry for uploading Docker image.
CREATE IMAGE REPOSITORY IF NOT EXISTS detokenize_repository;

-- Verify what's going on.
SHOW COMPUTE POOLS; 
SHOW WAREHOUSES;
SHOW IMAGE REPOSITORIES;

-- Check the compute pool state.
DESCRIBE COMPUTE POOL detokenize_compute_pool;

-- Create Go detokenize service
CREATE SERVICE detokenize_service
  IN COMPUTE POOL detokenize_compute_pool
  FROM SPECIFICATION $$
spec:
  containers:
    - name: detokenize
      image: <REPLACE_WITH_IMAGE_LOCAION>
      env:
        SNOWFLAKE_WAREHOUSE: detokenize_warehouse
  endpoints:
    - name: detokenize
      port: 8080
      public: true  $$
external_access_integrations = (ALLOW_BACKEND_INTEGRATION);
```

The code below shows how to create the UDF and call it.
``` sql
CREATE FUNCTION detokenize_udf (InputText varchar)
  RETURNS varchar
  SERVICE=detokenize_service
  ENDPOINT=detokenize
  AS '/detokenize';

USE ROLE test_role;

SELECT detokenize_udf(last_name) FROM snowflake_benchmarking;
```