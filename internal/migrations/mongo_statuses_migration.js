db.results.find( {'executionresult.status':'success'},  {'executionresult.status':1} ).forEach( function(r) {  return db.results.update({_id: r._id}, {$set: {'executionresult.status':'passed' }})  } )
db.results.find( {'executionresult.status':'pending'},  {'executionresult.status':1} ).forEach( function(r) {  return db.results.update({_id: r._id}, {$set: {'executionresult.status':'running' }})  } )
db.results.find( {'executionresult.status':'error'},  {'executionresult.status':1} ).forEach( function(r) {  return db.results.update({_id: r._id}, {$set: {'executionresult.status':'failed' }})  } )


db.testresults.find( {'status':'success'},  {'status':1} ).forEach( function(r) {  return db.testresults.update({_id: r._id}, {$set: {'status':'passed' }})  } )
db.testresults.find( {'status':'pending'},  {'status':1} ).forEach( function(r) {  return db.testresults.update({_id: r._id}, {$set: {'status':'running' }})  } )
db.testresults.find( {'status':'error'},  {'status':1} ).forEach( function(r) {  return db.testresults.update({_id: r._id}, {$set: {'status':'failed' }})  } )