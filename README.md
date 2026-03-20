# TTK4145
Repo for heislab-prosjektet i TTK4145 Sanntidsprogrammering, et SW-prosjekt som skal betjene m etasjer ved hjelp av n heiser, og kravene til feiltoleranse er EKSTREME


## Overordnet beskrivelse av hva koden gjør
Koden implementerer et heissystem som er i stand til å kontrollere sin egen heis, og samhandle med n andre heiser for å fordele heisordre og sammen betjene m etasjer på en effektiv måte. Ulike feilmoduser på enkeltheiser, som at enkeltheiser får motorstopp, håndteres gjennom omfordeling av heisordre, slik at alle ordre betjenes. Heisene husker også hverandres lokale "cab calls", og gir disse til heiser som restartes og glemmer sine calls, slik at ingen ordre går tapt, selv om enkelte heisers programvare krasjer. 

Koden styrer den fysiske heisen gjennom et tcp-grensesnitt over localhost.


## Beskrivelse av hvordan kjøre koden 

Koden kompileres med 
```
go build root 
```
fra inne i grunnmappa "TTK4145", noe som genererer en root-fil (Linux) eller root.exe-fil (Windows) i samme mappe. 

For å kjøre koden, kaller man:

For Windows:
```
./root.exe -id=<valgfri id> -port=<port til hardware-server>
```
For Linux:
```
./root -id=<valgfri id> -port=<port til hardware-server>
```



## Litt beskrivelse av filstruktur og oversikt over alle moduler

Programmet avhenger i stor grad av goroutiner og kanaler for å utføre nødvendig sending/mottaking av nettverksmeldinger og å oppdage hardware-input, samtidig som heisen kjører.

Hovedmodulene kjøres som separate goroutiner som kommuniserer gjennom kanaler. Hovedmodulene er:


#### Elevator

Denne modulen implementerer en finite state machine som kontrollerer heisens bevegelser og dør. Den håndterer kun sine egne tildelte calls.

Modulen inneholder en Door tråd som styrer døra til heisen.


#### main

Initialiserer systemet og hoved-goroutinene, og fungerer som en koordinator mellom Sync og Elevator.


#### Sync

Den sentrale noden i hver enkelt heis for all informasjon som deles mellom heiser. Den bestemmer hvilken data som til enhver tid skal broadcastes til andre heiser, og videreformidler dette til Network. Sync oppdaterer heisens lagrede data basert på inputs (lokale knappetrykk/fullførte calls, melding om andre heiser sin state, eller meldinger om deres calls), merger data lokalt, og videreformidler oppdatert data til main og Network gjennom kanaler.

#### Network

Network-modulen håndterer all nettverkskommunikasjon i det distribuerte systemet.

Den håndterer peer discovery, status broadcasting, og uveksling av cab calls. Modulen videreformidler nettverksmeldinger til Sync-nivået, og etterspør lokal heisstatus for broadcasting.


#### elevio

elevio inneholder funksjoner for behandling av tcp-tilkoblingen og kommunikasjon til tcp hardware-serveren. Den inneholder funksjoner som kjøres som goroutiner for å overvåke sensor- og knappe-inputs, som etasje- og dørhindringssensoren, og har hjelpefunksjoner for å sette ting som motorretning og lysene på heisdisplayet.


#### sequenceassigner

Denne modulen brukes av main. Den mottar systemstatus, som inkluderer self- og peer-state, cab calls og bekreftede hall calls, og bruker et eksternt program til å tildele heisene hall calls.
sequenceassigner kjøres uavhengig på hver heis.

#### Lights

Lights er en goroutine som interagerer med hardware-serveren for å styre lysene. Den brukes av main.

#### elevstate

Elevstate inneholder enums og structs for å definere heistilstander.

## Naming conventions

This part is written in english, as the project uses english variable names.

The main naming conventions and names are the following:

The system contains many different datatypes and datastructures, as well as data from both the local elevator and its peers. To differentiate between local data and data from peers, the variable names have a prefix "self" or "peer".

"Cab calls" and "Hall calls" are the resulting orders/calls from pressing the button inside the elevator cab and on the floor panel, respectively.

Each elevator has a state represented by the ElevState type, which includes behaviour, direction, floor, and boolean flags for motor stop and door obstruction.

Elevators also share their own cab calls and their local version of the shared hall calls, in addition to their "State". This combination is called "Status".

Channel names are suffixed with "C". The name also contains information about what meaning the data it passes contains, like "peerStatusUpdate". If a channel passes info between modules, it also specifies the receiver of the channel, like "ToSync". Example:
peerStatusUpdateToSyncC

