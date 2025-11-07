# goratelimiter
tcp rate limiting service to centralize rate limiting accross multiple Application Replicas




## RESPONSES:


OK: 1  
MAX_USER_CONCURRENCY_REACHED: 2  
MAX_TOTAL_CONCURRENCY_REACHED: 3  
MAX_VOLUME_REACHED: 4  
INTERNAL_SERVER_ERROR: 5
UNKNOWN_STATUS: 0  

basically whenever a status other the OK is returned, the ratelimiter says that the user can't run the request


## CLIENT APIS

aquire: call to check if the user is allowed to run a given job at the moment
````
AQUIRE <usergroup> <max_usergroup_concurrency> <max_user_volume>
````

return: call when the given job is done
````
RETURN <usergroup>
````


## ADMIN APIS
reset all: call to set the states back to 0
````
RESET ALL
````
Increase TOTAL_CONCURRENCY: call to change the the max total concurrency
````
CONCURRENCY ADJUST <NEW_MAX_CONCURRENCY>
````








