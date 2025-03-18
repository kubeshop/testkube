from locust import HttpUser, task

class locust_example_test(HttpUser):
    @task
    def locust_example(self):
        self.client.get("/")
