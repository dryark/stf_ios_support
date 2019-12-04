## STF Server Setup
### Set environment variables
1. Follow root README to build server docker image
1. Get that image onto your server machine
1. Look through docker-compose.yml
1. For each used environment variable, set it properly in your system
    
    1. STF_IMAGE ( the tag of your docker image; if using example name: stf_with_ios:1.0 )
    1. STF_URI ( hostname of your server )
    1. PUBLIC_IP ( IP address of your server )
1. Setup certificates for Nginx on your local system
1. Pass the paths for those cert in by tweaking the mounted files in docker-compose.yml

    1. eg: Change the `/todo/yourpath/` parts
1. `docker-compose up`
