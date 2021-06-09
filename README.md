# manago
my own web app framework 

Not mentioned to open source, so no documentation and description :( 

## manago v2

Plan na wersję v2: uproszczoną, lżejszą. Będę wrzucał pomysły i co zachować/co wyrzucić

### new functionality

1. MUX - dodać blokowanie mux z poziomu `manago`, przydatne np. przy generowaniu nowych unikalnych nazw (iteracja). Do tego celu mógłby być dedykowany MUX eg 'NextNoGenerateMux`.

### major changes

1. Brak generowania `ctr` po nazwie - podawać do `Router` funkcje danego modelu, a te funkcje niech przyjmują `*Manago` w parametrze i zwracają przynajmniej `err`, resztę zrobić w obszarze `manago`
