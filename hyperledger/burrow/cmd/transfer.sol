pragma solidity >=0.4.0 <0.7.0;

contract Caller {
    function send(address payable x) public payable {
        x.transfer(msg.value);
    }
}