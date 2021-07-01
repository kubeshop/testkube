# KubeTest


# Components

- kubectl plugin - simple - installed w/o 3rd party repositories, communicates with  
- REST API Server - uses

## Some decisions: 

-  [ ] Which operator framework we shoul use for Controller - use "Operator Framework" or "Kubebuilder"
  + https://operatorframework.io/
  + https://book.kubebuilder.io/

  https://github.com/operator-framework/operator-sdk/issues/1758


- [ ]  Use postman, use inline postman collection definition in CRD yaml file.

- [ ] Golang REST API framework 
  + pure net/http
  + gofiber
  + ... or other

# Where to start

- [ ] CRD definition, CRD will hold content of Postman collection
  + Kind: KubeTest
  + Type: PostmanCollection
  + Content: inline json content

- We need to generate new Operator (with use of operator framework of our choice look above)

- Create basic REST API app (consider use plain go net/http package or small framework like gofiber or similiar) 




