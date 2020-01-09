## STF Server Setup
### Set environment variables
1. `docker pull openstf/stf`
1. Look through docker-compose.yml
1. Update `.env` with your environment settings
    
    1. STF_IMAGE ( custom image if desired otherwise openstf/stf )
    1. HOSTNAME ( hostname of your server )
    1. PUBLIC_IP ( IP address of your server )
1. Setup certificates for Nginx on your local system
1. Pass the paths for those cert in by tweaking the mounted files in docker-compose.yml

    1. eg: Change the `/todo/yourpath/` parts
1. `docker-compose up`
