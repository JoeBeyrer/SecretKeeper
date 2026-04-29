// before: Alice sends Bob a friend request so Bob always has an incoming
// request to view — no conditional branching needed.
describe('friends_view_requests', () => {
  before(() => {
    cy.task('clearDB');
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');
    cy.contains('.tab', 'Add Friend').click();
    cy.get('.add-section .field-input').type('Bob');
    cy.get('.send-btn').click();
    cy.get('.feedback').should('be.visible');
  });

  it('Bob sees the incoming request on the Requests tab', () => {
    cy.login('Bob', 'Bob12345');
    cy.visit('/friends');
    cy.contains('.tab', 'Requests').click();
    cy.get('.friend-row').should('have.length.at.least', 1);
    cy.get('.friend-row').first().should('be.visible');
  });
});
