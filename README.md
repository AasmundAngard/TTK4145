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

Lurt å nevne at alt spawnes i main kanskje?

Programmet avhenger i stor grad på goroutiner og kanaler for å utføre nødvendig sending/mottaking av nettverksmeldinger, og for å oppdage hardware-input.

Hovedmodulene kjøres som separate goroutiner som kommuniserer gjennom kanaler. Hovedmodulene er:

The program relies heavily on goroutines and channels to perform the necessary sending/receiving of network messages, and to detect hardware input.

The main modules are ran as separate goroutines, communicating through channels. The main modules are
#### Elevator
Implements a finite state machine that controls the elevator movement and door. Only has knowledge of its own assigned calls.

Contains a Door thread that controls the door of the elevator.

#### main
Initializes the system and the main goroutines, and acts as a coordinator between Sync and Elevator.

#### Sync

Central node of all shared information in the system. Decides what data to broadcast to other elevators, and passes this to Network. Updates stored data based on inputs (local button presses, completed calls, incoming elevator states from itself and peers, and calls from peers), and passes updated data to main and Network through channels.

#### Network

Network handles the connections and communications between elevators, and monitors the ports for incoming updates or requests from other elevators.

Network consists of multiple threads, each listening on one port, or used for handling specific requests from Sync.

## Naming conventions

The system contains many different datatypes and datastructures, as well as data from both the local elevator and its peers.

To differentiate between local data and data from peers, the variable names have a prefix "self" or "peer".

The elevators have a "State" based on the type Elevstate, containing Behaviour, Direction and Floor.

Elevators also share their own cab calls and their local version of the shared hall calls, in addition to their "State". This combination is called "Status".

Channel names are suffixed with "C". The name also contains information about what meaning the data it passes contains, like "peerStatusUpdate". If a channel passes info between modules, it also specifies the receiver of the channel, like "ToSync". Example:
peerStatusUpdateToSyncC

